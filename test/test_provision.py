import datetime
import json
import os
import subprocess
import time
import unittest
import uuid

import requests.auth


class TestProvision(unittest.TestCase):
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

    def test_provision(self):
        url = self.host + "/v2/service_instances/{}?accepts_incomplete=true".format(self.instance_id)
        r = requests.put(url, auth=self.auth, headers=self.headers, data=json.dumps({
            "service_id": self.service_id,
            "plan_id": self.plan_id,
        }))
        self.assertEqual(202, r.status_code)

        start_time = datetime.datetime.now()
        diff = datetime.timedelta(seconds=0)
        state = "in progress"
        while state == "in progress" and diff < datetime.timedelta(minutes = 2):
            time.sleep(5)
            url = self.host + "/v2/service_instances/{}/last_operation?operation=provision&service_id={}&plan_id={}".format(
                self.instance_id, self.service_id, self.plan_id
            )
            r = requests.get(url, auth=self.auth, headers=self.headers, data=json.dumps({
                "service_id": self.service_id,
                "plan_id": self.plan_id,
            }))
            self.assertEqual(200, r.status_code)
            json_body = json.loads(r.content.decode())
            state = json_body["state"]
            print("provisioning in progress, state: {}...".format(state))
            now_time = datetime.datetime.now()
            diff = now_time - start_time

        self.assertGreater(datetime.timedelta(minutes = 2), diff, "timed out while waiting for service to report success")
        self.assertEqual("succeeded", state)

        cmd = "kubectl get namespace kibosh-{} -o json".format(self.instance_id)
        self.run_command(cmd)

        cmd = "kubectl get pods --namespace kibosh-{} -o json".format(self.instance_id)
        json_body = self.run_command(cmd)
        self.assertEqual(1, len(json_body["items"]))
        print(json_body["items"][0]["status"]["conditions"])

        cmd = "kubectl get services --namespace kibosh-{} -o json".format(self.instance_id)
        json_body = self.run_command(cmd)
        self.assertEqual(1, len(json_body["items"]))
        self.assertEqual(1, len(json_body["items"][0]["status"]["loadBalancer"]["ingress"]))

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
