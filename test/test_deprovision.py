import datetime
import time

import requests

from test_broker_base import TestBrokerBase


class TestDeprovision(TestBrokerBase):
    def test_deprovision(self):
        path = "/v2/service_instances/{}?service_id={}&plan_id={}".format(
            self.instance_id, self.service_id, self.plan_id
        )

        delete_response = self.call_broker(path, {}, requests.delete)
        self.assertIn("operation", delete_response)
        self.assertEqual(delete_response["operation"], "deprovision")

        start_time = datetime.datetime.now()
        diff = datetime.timedelta(seconds=0)
        state = "in progress"
        while state == "in progress" and diff < datetime.timedelta(minutes=2):
            print("deprovisioning in progress, state: {}...".format(state))
            time.sleep(5)
            path = "/v2/service_instances/{}/last_operation?operation=deprovision&service_id={}&plan_id={}".format(
                self.instance_id, self.service_id, self.plan_id)
            delete_status = self.call_broker(path, {}, requests.get)

            state = delete_status["state"]
            now_time = datetime.datetime.now()
            diff = now_time - start_time

        self.assertGreater(datetime.timedelta(minutes=2), diff, "timed out waiting for service to report deletion")
        self.assertEqual("succeeded", state)

        start_time = datetime.datetime.now()
        diff = datetime.timedelta(seconds=0)
        namespace_names = ["kibosh-".format(self.instance_id)]
        while diff < datetime.timedelta(minutes=1) and "kibosh-".format(self.instance_id) in namespace_names:
            print("deleting namespace in progress")
            time.sleep(5)
            cmd = "kubectl get namespaces -o json".format(self.instance_id)
            get_namespace_json = self.run_command(cmd)

            namespace_names = [n["metadata"]["name"] for n in get_namespace_json["items"]]

            now_time = datetime.datetime.now()
            diff = now_time - start_time

        self.assertGreater(datetime.timedelta(minutes=2), diff, "timed out waiting for namespace to delete")
