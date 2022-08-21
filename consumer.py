
from pulsar import InitialPosition
import pulsar
import pprint
client = pulsar.Client('pulsar://localhost:6650')
consumer = client.subscribe(topic=["persistent://terra/localterra/tm.event='NewBlock'", "persistent://terra/localterra/tm.event='Tx'"],
                            subscription_name='my-subscription', initial_position=InitialPosition.Latest)
while True:
    msg = consumer.receive()
    try:
        print("Received message '{}' id='{}'".format(
              msg.data(), msg.message_id()))
        # Acknowledge successful processing of the message
        consumer.acknowledge(msg)
    except Exception:
        # Message failed to be processed
        consumer.negative_acknowledge(msg)

client.close()
