import csv
import datetime
import time


""" Bets storage location. """
STORAGE_FILEPATH = "./bets.csv"
""" Simulated winner number in the lottery contest. """
LOTTERY_WINNER_NUMBER = 7574


""" A lottery bet registry. """
class Bet:
    def __init__(self, agency: str, first_name: str, last_name: str, document: str, birthdate: str, number: str):
        """
        agency must be passed with integer format.
        birthdate must be passed with format: 'YYYY-MM-DD'.
        number must be passed with integer format.
        """
        self.agency = int(agency)
        self.first_name = first_name
        self.last_name = last_name
        self.document = document
        self.birthdate = datetime.date.fromisoformat(birthdate)
        self.number = int(number)

""" Checks whether a bet won the prize or not. """
def has_won(bet: Bet) -> bool:
    return bet.number == LOTTERY_WINNER_NUMBER

"""
Persist the information of each bet in the STORAGE_FILEPATH file.
Not thread-safe/process-safe.
"""
def store_bets(bets: list[Bet]) -> None:
    with open(STORAGE_FILEPATH, 'a+') as file:
        writer = csv.writer(file, quoting=csv.QUOTE_MINIMAL)
        for bet in bets:
            writer.writerow([bet.agency, bet.first_name, bet.last_name,
                             bet.document, bet.birthdate, bet.number])

"""
Loads the information all the bets in the STORAGE_FILEPATH file.
Not thread-safe/process-safe.
"""
def load_bets() -> list[Bet]:
    with open(STORAGE_FILEPATH, 'r') as file:
        reader = csv.reader(file, quoting=csv.QUOTE_MINIMAL)
        for row in reader:
            yield Bet(row[0], row[1], row[2], row[3], row[4], row[5])

"""
Parses a string to a Bet object.
Example of string: "[AgencyID:000,ID:7577,Name:SantiagoLionel,Surname:Lorca,PersonalID:30904465,BirthDate:1999-03-17]"
Agency is the number of the client.
"""
def parse_bet(bet_str: str) -> Bet:
    bet_data = dict([pair.split(":") for pair in bet_str.split(",")])
    return Bet(bet_data["AgencyID"], bet_data["Name"], bet_data["Surname"], bet_data["PersonalID"], bet_data["BirthDate"], bet_data["ID"])

"""
Parses a string to a list of Bet objects.
Example of string: "Bets -> [bet1][bet2][bet3]"
"""
def parse_bets(bets_str: str) -> list[Bet]:
    bets_str = bets_str.split("Bets -> ")
    if len(bets_str) != 2:
        return []
    bets_str = bets_str[1][1:-1]
    return [parse_bet(bet_str) for bet_str in bets_str.split("][")]