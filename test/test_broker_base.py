import os
import uuid

import requests.auth


class TestBrokerBase():
    @classmethod
    def setUpClass(self):
        os.environ['BROKER_HOST'] = str("http://localhost:8090")
        os.environ['BROKER_USERNAME'] = str("admin")
        os.environ['BROKER_PASSWORD'] = str("password")

        self.host = os.environ['BROKER_HOST']
        self.username = os.environ['BROKER_USERNAME']
        self.password = os.environ['BROKER_PASSWORD']

        self.auth = requests.auth.HTTPBasicAuth(self.username, self.password)
        self.headers = {
            "X-Broker-API-Version": "2.14",
        }

        self.instance_id = uuid.uuid4()
        self.service_id = "c76ed0a4-9a04-5710-90c2-75e955697b08"
        self.plan_id = self.service_id + "-small"