
from pulsar import MessageId
import pulsar
import pprint
import re
client = pulsar.Client('pulsar://localhost:6650')
msg_id = pulsar.MessageId.earliest
reader_block = client.create_reader(
    topic="persistent://terra/localterra/tm.event='NewBlock'", start_message_id=msg_id)
reader_tx = client.create_reader(
    topic="persistent://terra/localterra/tm.event='Tx'", start_message_id=msg_id)
while True:
    msg = reader_block.read_next()
    print("Received NewBlock message '{}' id='{}'".format(
        msg.data(), msg.message_id()))
    msg = reader_tx.read_next()
    print("Received NewTx message '{}' id='{}'".format(
        msg.data(), msg.message_id()))

    # No acknowledgment
