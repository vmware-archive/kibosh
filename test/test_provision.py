import datetime
import json
import time

import requests.auth

from test_broker_base import TestBrokerBase


class TestProvision(TestBrokerBase):
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
        while state == "in progress" and diff < datetime.timedelta(minutes=2):
            print("provisioning in progress, state: {}...".format(state))
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
            now_time = datetime.datetime.now()
            diff = now_time - start_time

        self.assertGreater(datetime.timedelta(minutes=2), diff, "timed out while waiting for service to report success")
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
