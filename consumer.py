
from pulsar import InitialPosition
import pulsar
import pprint
client = pulsar.Client('pulsar://localhost:6650')
consumer = client.subscribe(
    'localterra', 'my-subscription', initial_position=InitialPosition.Latest)
while True:
    msg = consumer.receive()
    try:
        # print("Received message '{}' id='{}'".format(
        #    msg.data(), msg.message_id()))
        pprint.pprint(msg)
        # Acknowledge successful processing of the message
        consumer.acknowledge(msg)
    except Exception:
        # Message failed to be processed
        consumer.negative_acknowledge(msg)

client.close()
