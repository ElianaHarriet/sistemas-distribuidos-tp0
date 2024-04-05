import socket
import logging
from common.utils import parse_bets, store_bets, load_bets, has_won
from multiprocessing import Process, Manager, Lock, Semaphore
from os import kill
from signal import SIGTERM

AGENCIES_DONE = "AGENCIES_DONE"
SAVE_BETS = "SAVE_BETS"
MAX_TRIES = 10
MAX_THREADS = 5 # In this case is the number of agencies


class Server:
    def __init__(self, port, listen_backlog, n_agencies):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._n_agencies = n_agencies
        self._manager = Manager()
        self._agencies_done = {}

    def run(self):
        with self._manager as manager:
            self._connections = manager.list()  # shared list
            self._processes = manager.list()  # shared list
            self._agencies_done = manager.dict()  # shared dict
            self._winners = manager.list()  # shared list
            locks = {AGENCIES_DONE: Lock(), SAVE_BETS: Lock()}
            semaphore = Semaphore(MAX_THREADS)

            for i in range(self._n_agencies):
                self._agencies_done[i + 1] = False

            while True:
                semaphore.acquire()
                client_sock = self.__accept_new_connection()
                p = Process(target=self.__handle_client_connection, args=(client_sock.fileno(), locks, semaphore))
                p.start()
                self._processes.append(p.pid)

    def __handle_client_connection(self, client_sock_fd, locks, semaphore):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        client_sock = socket.fromfd(client_sock_fd, socket.AF_INET, socket.SOCK_STREAM)
        self._connections.append(client_sock.fileno())
        msg_buffer = ""
        msg, msg_buffer = self.__receive_line(client_sock, msg_buffer)
        not_break = True
        personal_ids = set()
        while msg and not_break:
            if "Bets" in msg:
                not_break = self.__manage_new_bets(msg, client_sock, personal_ids, locks[SAVE_BETS])
            elif "Awaiting results" in msg:
                agency = self.__get_agency_from_msg(msg)
                not_break = self.__manage_results(client_sock, agency, personal_ids, locks[AGENCIES_DONE], locks[SAVE_BETS])
            else:
                self.__send_message(client_sock, "ERROR: Mensaje no reconocido")
            msg, msg_buffer = self.__receive_line(client_sock, msg_buffer)
        self.__close_client_connection(client_sock.fileno())
        semaphore.release()
            

    def __manage_new_bets(self, msg, client_sock, personal_ids, save_bets_lock):
        try:
            bets = parse_bets(msg)
        except Exception as e:
            logging.error(f'action: parse_bets | result: fail | error: {e} | msg: {msg}')
            self.__send_message(client_sock, "ERROR: Error al parsear las apuestas")
            return False
        self.__send_message(client_sock, f"OK: Apuestas recibidas | Cantidad:{len(bets)}")
        save_bets_lock.acquire()
        store_bets(bets)
        for bet in bets:
            personal_ids.add(bet.document)
        save_bets_lock.release()
        for bet in bets:
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
        return True

    def __manage_results(self, client_sock, agency, personal_ids, agencies_done_lock, save_bets_lock):
        agencies_done_lock.acquire()
        self._agencies_done[agency] = True
        all_done = self.__all_agencies_done()
        agencies_done_lock.release()
        if not all_done:
            self.__send_message(client_sock, "WAIT: Esperando a las otras agencias")
            return True
        save_bets_lock.acquire()
        winners = self.__get_winners()
        agency_winners = []
        for winner in winners:
            if winner in personal_ids:
                agency_winners.append(winner)
        save_bets_lock.release()
        agency_winners = ','.join(agency_winners)
        self.__send_message(client_sock, f"OK: Sorteo realizado | Ganadores:{agency_winners}")
        return False

    def __receive_line(self, client_sock, msg_buffer):
        """
        Read a line message from a specific client socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        
        It also avoids short-writes
        """
        try:
            tries = 0
            while not "\n" in msg_buffer:
                tries += 1
                chunk = client_sock.recv(1024)
                if not chunk and tries == MAX_TRIES:
                    logging.error(f'action: receive_message | result: fail | error: connection closed')
                    return None, msg_buffer
                elif not chunk:
                    continue
                msg_buffer += chunk.decode('utf-8')
            msg, _, msg_buffer = msg_buffer.partition("\n")
            logging.info(f'action: receive_message | result: success | ip: {client_sock.getpeername()[0]} | msg: {msg}')
            return msg.rstrip(), msg_buffer
        except OSError as e:
            logging.error(f'action: receive_message | result: fail | error: {e}')
            return None, msg_buffer
    
    def __send_message(self, client_sock, msg):
        """
        Send a message to a specific client socket

        If a problem arises in the communication with the client.
        Then the client socket will also be closed

        It also avoids short-reads
        """
        msg = "{}\n".format(msg).encode('utf-8')
        total_sent = 0
        while total_sent < len(msg):
            sent = client_sock.send(msg[total_sent:])
            if sent == 0:
                logging.error(f'action: send_message | result: fail | error: connection closed')
            total_sent = total_sent + sent

    def __close_client_connection(self, client_sock_fd):
        """
        Closes the client socket and removes it from the connections set
        """
        self._connections.remove(client_sock_fd)
        client_sock = socket.fromfd(client_sock_fd, socket.AF_INET, socket.SOCK_STREAM) 
        client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def __all_agencies_done(self):
        """
        Checks if all agencies have finished sending bets
        """
        for agency in self._agencies_done.keys():
            if not self._agencies_done[agency]:
                return False
        return True
    
    def __get_agency_from_msg(self, msg):
        """
        Extracts the agency number from a message
        Format: '[Client1] Awaiting results'
        """
        client_id = msg.split(']')[0]
        client_id = client_id.split(' ')[1]
        return int(client_id)
    
    def __get_winners(self):
        """
        Returns the winners of the lottery
        """
        if not self._winners:
            for bet in load_bets():
                if has_won(bet):
                    self._winners.append(bet.document)
            logging.info(f'action: sorteo | result: success | cant_ganadores: {len(self._winners)}')
        return self._winners
    
    def stop(self):
        """
        Closes all the file descriptors and sockets
        """
        for pid in self._processes:
            kill(pid, SIGTERM)
        for client_sock_fd in self._connections:
            client_sock = socket.fromfd(client_sock_fd, socket.AF_INET, socket.SOCK_STREAM) 
            client_sock.shutdown(socket.SHUT_RDWR)
            client_sock.close()
        self._server_socket.close()
