import json
import os
import unittest

import requests.auth

from test_broker_base import TestBrokerBase


class TestCatalog(TestBrokerBase, unittest.TestCase):
    def test_catalog(self):
        url = self.host + "/v2/catalog"
        r = requests.get(url, auth=self.auth, headers=self.headers)
        self.assertEqual(200, r.status_code)
        json_body = json.loads(r.content.decode())
        self.assertEqual(1, len(json_body["services"]))
        self.assertEqual("c76ed0a4-9a04-5710-90c2-75e955697b08", json_body["services"][0]["id"])
        self.assertEqual("mysql", json_body["services"][0]["name"])
        self.assertEqual(2, len(json_body["services"][0]["plans"]))