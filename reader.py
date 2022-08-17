
from pulsar import MessageId
import pulsar
import pprint
import re
client = pulsar.Client('pulsar://localhost:6650')
msg_id = pulsar.MessageId.earliest
rx = re.compile('persistent://public/default/.*')
reader = client.create_reader(topic='localterra', start_message_id=msg_id)

while True:
    msg = reader.read_next()
    pprint.pprint(msg.message_id())
    pprint.pprint(msg.data())

    #print("Received message '{}' id='{}'".format(msg.data(), msg.message_id()))
    # No acknowledgment
