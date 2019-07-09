import uuid

import mysql.connector
import requests

from test_broker_base import TestBrokerBase


class TestBindUnbind(TestBrokerBase):
    @classmethod
    def setUpClass(self):
        super().setUpClass()
        self.binding_id = uuid.uuid4()

    def call_bind(self):
        path = "/v2/service_instances/{}/service_bindings/{}".format(self.instance_id, self.binding_id)
        return self.call_broker(path, {
            "service_id": self.service_id,
            "plan_id": self.plan_id,
        }, requests.put)

    def test_bind_response_credentials(self):
        json_body = self.call_bind()

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

    def test_unbind_response(self):
        path = "/v2/service_instances/{}/service_bindings/{}?service_id={}&plan_id={}".format(
            self.instance_id, self.binding_id, self.service_id, self.plan_id,
        )
        delete_body = self.call_broker(path, {}, requests.delete)

        self.assertEqual({}, delete_body)
