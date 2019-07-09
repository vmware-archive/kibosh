import json
import os
import subprocess
import unittest

import requests.auth


class TestBrokerBase(unittest.TestCase):
    instance_id = ""

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

        self.service_id = "c76ed0a4-9a04-5710-90c2-75e955697b08"
        self.plan_id = self.service_id + "-small"

    def run_command(self, cmd):
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)
        out, err = p.communicate()
        if p.returncode != 0:
            print()
            print(cmd)
            print(out.decode("utf8"))
            print(err.decode("utf8"))
        self.assertEqual(0, p.returncode)
        return json.loads(out)

    def call_broker(self, path: str, body: dict):
        url = self.host + path
        r = requests.put(url, auth=self.auth, headers=self.headers, data=json.dumps(body))
        self.assertGreater(400, r.status_code)

        return json.loads(r.content.decode())
