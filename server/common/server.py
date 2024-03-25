import socket
import logging
from common.utils import parse_bets, store_bets, load_bets, has_won


class Server:
    def __init__(self, port, listen_backlog, n_agencies):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._connections = set()
        self._agencies_done = {}
        for i in range(n_agencies):
            self._agencies_done[i + 1] = False
        self._winners = None

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        while True:
            client_sock = self.__accept_new_connection()
            self.__handle_client_connection(client_sock)

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        msg = self.__receive_line(client_sock)
        if msg is None:
            return
        if "Bets" in msg:
            self.__manage_new_bets(msg, client_sock)
        elif "Awaiting results" in msg:
            agency = self.__get_agency_from_msg(msg)
            self.__manage_results(client_sock, agency)
        else:
            self.__send_message(client_sock, "ERROR: Mensaje no reconocido")
        self.__close_client_connection(client_sock)

    def __manage_new_bets(self, msg, client_sock):
        try:
            bets = parse_bets(msg)
        except Exception as e:
            logging.error(f'action: parse_bets | result: fail | error: {e}')
            self.__send_message(client_sock, "ERROR: Error al parsear las apuestas")
            self.__close_client_connection(client_sock)
            return
        self.__send_message(client_sock, f"OK: Apuestas recibidas | Cantidad:{len(bets)}")
        store_bets(bets)
        for bet in bets:
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')

    def __manage_results(self, client_sock, agency):
        self._agencies_done[agency] = True
        if not self.__all_agencies_done():
            self.__send_message(client_sock, "WAIT: Esperando a las otras agencias")
            return
        winners = self.__get_winners()
        winners = ','.join(winners)
        self.__send_message(client_sock, f"OK: Sorteo realizado | Ganadores:{winners}")

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

    def __close_client_connection(self, client_sock):
        """
        Closes the client socket and removes it from the connections set
        """
        client_sock.close()
        self._connections.remove(client_sock)

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        self._connections.add(c)
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def __all_agencies_done(self):
        """
        Checks if all agencies have finished sending bets
        """
        for agency in self._agencies_done:
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
            bets = [bet for bet in load_bets() if has_won(bet)]
            self._winners = [bet.document for bet in bets]
            logging.info(f'action: sorteo | result: success | cant_ganadores: {len(self._winners)}')
        return self._winners
    
    def stop(self):
        """
        Closes all the file descriptors and sockets
        """
        while self._connections:
            conn = self._connections.pop()
            conn.close()
        self._server_socket.close()
