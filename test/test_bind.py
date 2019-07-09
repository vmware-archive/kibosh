import json
import uuid

import mysql.connector
import requests

from test_broker_base import TestBrokerBase


class TestBind(TestBrokerBase):
    @classmethod
    def setUpClass(self):
        super().setUpClass()
        self.binding_id = uuid.uuid4()

    def call_bind(self):
        # TODO: HELPER FOR BROKER AUTH
        url = self.host + "/v2/service_instances/{}/service_bindings/{}".format(self.instance_id, self.binding_id)
        r = requests.put(url, auth=self.auth, headers=self.headers, data=json.dumps({
            "service_id": self.service_id,
            "plan_id": self.plan_id,
        }))
        self.assertEqual(201, r.status_code)

        return json.loads(r.content.decode())

    def test_bind_response_credentials(self):
        json_body = self.call_bind()

        print(json.dumps(json_body, indent=2))
        self.assertIn("credentials", json_body)

    def test_bind_template(self):
        json_body = self.call_bind()
        credentials = json_body["credentials"]

        self.assertIn("hostname", credentials)
        self.assertIn("jdbcUrl", credentials)
        self.assertIn("name", credentials)
        self.assertIn("password", credentials)
        self.assertEqual(credentials["port"], 3306)
        self.assertIn("uri", credentials)
        self.assertEqual(credentials["username"], "root")

    def test_connection(self):
        json_body = self.call_bind()
        credentials = json_body["credentials"]
        user = credentials["username"]
        password = credentials["password"]
        host = credentials["hostname"]
        port = credentials["port"]

        cnx = mysql.connector.connect(
            user=user, password=password, host=host, port=port
        )

        cnx.close()
