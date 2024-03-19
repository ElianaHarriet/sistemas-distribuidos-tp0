"""
Parses a string to a Bet object.
Example of string: "[CLIENT 1] Bet -> [ID: 7577, Name: Santiago Lionel, Surname: Lorca, PersonalID: 30904465, BirthDate: 1999-03-17]"
"""
def parse_bet(agency: str, bet_str: str):
    bet_str = bet_str.replace(" ", "")
    bet_data = bet_str.split("->")[1].strip()[1:-1]
    # bet_data to dictionary
    bet_data = dict([pair.split(":") for pair in bet_data.split(",")])
    bet_data["ID"] = int(bet_data["ID"])
    bet_data["PersonalID"] = int(bet_data["PersonalID"])
    return bet_data

print(parse_bet("hola", "[CLIENT 1] Bet -> [ID: 7577, Name: Santiago Lionel, Surname: Lorca, PersonalID: 30904465, BirthDate: 1999-03-17]"))