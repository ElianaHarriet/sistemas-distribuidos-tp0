import socket
import logging
from common.utils import parse_bet, store_bets


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._connections = set()

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
        bets = parse_bets(msg)
        self.__send_message(client_sock, f"Apuestas recibidas | Cantidad: {len(bets)}")
        self.__close_client_connection(client_sock)
        store_bets(bets)
        # logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')

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
                if message.endswith(b'\n'):
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
    
    def stop(self):
        """
        Closes all the file descriptors and sockets
        """
        while self._connections:
            conn = self._connections.pop()
            conn.close()
        self._server_socket.close()
