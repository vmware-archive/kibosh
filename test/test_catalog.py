import json

import requests.auth

from test_broker_base import TestBrokerBase


class TestCatalog(TestBrokerBase):
    def test_catalog(self):
        path = "/v2/catalog"
        catalog_json = self.call_broker(path, {}, requests.get)

        self.assertEqual(1, len(catalog_json["services"]))
        self.assertEqual("c76ed0a4-9a04-5710-90c2-75e955697b08", catalog_json["services"][0]["id"])
        self.assertEqual("mysql", catalog_json["services"][0]["name"])
        self.assertEqual(2, len(catalog_json["services"][0]["plans"]))
