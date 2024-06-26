from common.server import Server
import logging
import signal
import sys
class ServerManager:
    def __init__(self, port, listen_backlog, n_agencies):
        self.server = Server(port, listen_backlog, n_agencies)

    def run(self):
        # Register the signal handler
        signal.signal(signal.SIGTERM, self.sigterm_handler)
        self.server.run()

    def sigterm_handler(self, signum, frame):
        logging.info("SIGTERM received, stopping server...")
        self.server.stop()