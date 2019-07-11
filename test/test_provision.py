import datetime
import time

import requests.auth

from test_broker_base import TestBrokerBase


class TestProvision(TestBrokerBase):
    def test_provision(self):
        path = "/v2/service_instances/{}?accepts_incomplete=true".format(self.instance_id)
        body = {
            "service_id": self.service_id,
            "plan_id": self.plan_id,
        }

        create_response = self.call_broker(path, body, requests.put)
        self.assertIn("operation", create_response)
        self.assertEqual(create_response["operation"], "provision")

        start_time = datetime.datetime.now()
        diff = datetime.timedelta(seconds=0)
        state = "in progress"
        while state == "in progress" and diff < datetime.timedelta(minutes=2):
            print("provisioning in progress, state: {}...".format(state))
            time.sleep(5)
            path = "/v2/service_instances/{}/last_operation?operation=provision&service_id={}&plan_id={}".format(
                self.instance_id, self.service_id, self.plan_id)
            create_status = self.call_broker(path, {}, requests.get)

            state = create_status["state"]
            now_time = datetime.datetime.now()
            diff = now_time - start_time

        self.assertGreater(datetime.timedelta(minutes=2), diff, "timed out while waiting for service to report success")
        self.assertEqual("succeeded", state)

        cmd = "kubectl get namespace kibosh-{} -o json".format(self.instance_id)
        self.run_command(cmd)

        cmd = "kubectl get pods --namespace kibosh-{} -o json".format(self.instance_id)
        get_pods_json = self.run_command(cmd)
        self.assertEqual(1, len(get_pods_json["items"]))

        cmd = "kubectl get services --namespace kibosh-{} -o json".format(self.instance_id)
        get_service_json = self.run_command(cmd)
        self.assertEqual(1, len(get_service_json["items"]))
        self.assertEqual(1, len(get_service_json["items"][0]["status"]["loadBalancer"]["ingress"]))
