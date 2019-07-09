import json
import os
import subprocess
import unittest

import requests.auth


class TestBrokerBase(unittest.TestCase):
    instance_id = ""

    @classmethod
    def setUpClass(self):
        self.host = os.getenv('BROKER_HOST', "http://localhost:8090")
        self.username = os.getenv('BROKER_USERNAME', "admin")
        self.password = os.getenv('BROKER_PASSWORD', "password")

        self.auth = requests.auth.HTTPBasicAuth(self.username, self.password)

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

    def call_broker(self, path: str, body: dict, requests_method):
        headers = {
            "X-Broker-API-Version": "2.14",
        }

        url = self.host + path
        r = requests_method(url, auth=self.auth, headers=headers, data=json.dumps(body))
        self.assertGreater(400, r.status_code)

        return json.loads(r.content.decode())
