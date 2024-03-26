import socket
import logging
from common.utils import parse_bets, store_bets, load_bets, has_won
from multiprocessing import Process, Manager, Lock
from os import kill
from signal import SIGTERM

AGENCIES_DONE = "AGENCIES_DONE"
WINNERS = "WINNERS"
SAVE_BETS = "SAVE_BETS"


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
            self._connections_lock = Lock()
            self._processes = manager.list()  # shared list
            self._processes_lock = Lock()
            self._agencies_done = manager.dict()  # shared dict
            self._winners = manager.list()  # shared list
            locks = {AGENCIES_DONE: Lock(), WINNERS: Lock(), SAVE_BETS: Lock()}

            for i in range(self._n_agencies):
                self._agencies_done[i + 1] = False

            while True:
                client_sock = self.__accept_new_connection()
                p = Process(target=self.__handle_client_connection, args=(client_sock.fileno(), locks))
                p.start()
                self._processes_lock.acquire()
                self._processes.append(p.pid)
                self._processes_lock.release()

    def __handle_client_connection(self, client_sock_fd, locks):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        client_sock = socket.fromfd(client_sock_fd, socket.AF_INET, socket.SOCK_STREAM)
        self._connections_lock.acquire()
        self._connections.append(client_sock.fileno())
        self._connections_lock.release()
        msg = self.__receive_line(client_sock)
        if msg is None:
            return
        if "Bets" in msg:
            self.__manage_new_bets(msg, client_sock, locks[SAVE_BETS])
        elif "Awaiting results" in msg:
            agency = self.__get_agency_from_msg(msg)
            self.__manage_results(client_sock, agency, locks[AGENCIES_DONE], locks[WINNERS])
        else:
            self.__send_message(client_sock, "ERROR: Mensaje no reconocido")

    def __manage_new_bets(self, msg, client_sock, save_bets_lock):
        try:
            bets = parse_bets(msg)
        except Exception as e:
            logging.error(f'action: parse_bets | result: fail | error: {e} | msg: {msg}')
            self.__send_message(client_sock, "ERROR: Error al parsear las apuestas")
            self.__close_client_connection(client_sock.fileno())
            return
        self.__send_message(client_sock, f"OK: Apuestas recibidas | Cantidad:{len(bets)}")
        self.__close_client_connection(client_sock.fileno())
        save_bets_lock.acquire()
        store_bets(bets)
        save_bets_lock.release()
        for bet in bets:
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')

    def __manage_results(self, client_sock, agency, agencies_done_lock, winners_lock):
        agencies_done_lock.acquire()
        self._agencies_done[agency] = True
        all_done = self.__all_agencies_done()
        agencies_done_lock.release()
        if not all_done:
            self.__send_message(client_sock, "WAIT: Esperando a las otras agencias")
            self.__close_client_connection(client_sock.fileno())
            return
        winners_lock.acquire()
        winners = self.__get_winners()
        winners_lock.release()
        winners = ','.join(winners)
        self.__send_message(client_sock, f"OK: Sorteo realizado | Ganadores:{winners}")
        self.__close_client_connection(client_sock.fileno())

    def __receive_line(self, client_sock):
        """
        Read a line message from a specific client socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        
        It also avoids short-writes
        """
        try:
            length = 0
            message = b''
            while True:
                chunk = client_sock.recv(1024)
                if not chunk:
                    break
                message += chunk
                length += len(chunk)
                if message.endswith(b'\n'): # this will cause no problems because every client send only one message ending in '\n'
                    break
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {message}')
            return message.decode('utf-8').rstrip()
        except OSError as e:
            logging.error(f'action: receive_message | result: fail | error: {e}')
            client_sock.close()
            self._connections.remove(client_sock)
            return None
    
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
        self._connections_lock.acquire()
        self._connections.remove(client_sock_fd)
        self._connections_lock.release()
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
        self._processes_lock.acquire()
        for pid in self._processes:
            kill(pid, SIGTERM)
        self._processes_lock.release()
        self._connections_lock.acquire()
        for client_sock_fd in self._connections:
            client_sock = socket.fromfd(client_sock_fd, socket.AF_INET, socket.SOCK_STREAM) 
            client_sock.shutdown(socket.SHUT_RDWR)
            client_sock.close()
        self._connections_lock.release()
        self._server_socket.close()
