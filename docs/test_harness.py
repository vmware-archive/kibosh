import json
import requests.auth
import sys

host = "http://localhost:8090"
username = "admin"
password = "monkey123"

instance_id = "tuesday3"
service_id = "c76ed0a4-9a04-5710-90c2-75e955697b08"
binding_id = "b49e7fe8-707b-499d-90b8-3ac0fb897421"
plan_id = service_id + "-small"

auth = requests.auth.HTTPBasicAuth(username, password)
headers = {
        "X-Broker-API-Version": "2.14",
}


def catalog():
    url = host + "/v2/catalog"
    try:
        print("Requesting url [{}]".format(url))
        r = requests.get(url, auth=auth, headers=headers)
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


def provision():
    url = host + "/v2/service_instances/{}?accepts_incomplete=true".format(instance_id)
    try:
        print("Requesting url [{}]".format(url))
        r = requests.put(url, auth=auth, headers=headers, data=json.dumps({
            "service_id": service_id,
            "plan_id": plan_id,
        }))
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


def provision_status():
    url = host + "/v2/service_instances/{}/last_operation?operation=provision&service_id={}&plan_id={}".format(
        instance_id, service_id, plan_id
    )
    try:
        print("Requesting url [{}]".format(url))
        r = requests.get(url, auth=auth, headers=headers, data=json.dumps({
            "service_id": service_id,
            "plan_id": plan_id,
        }))
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


def bind():
    url = host + "/v2/service_instances/{}/service_bindings/{}".format(instance_id, binding_id)
    try:
        print("Requesting url [{}]".format(url))
        r = requests.put(url, auth=auth, headers=headers, data=json.dumps({
            "service_id": service_id,
            "plan_id": plan_id,
        }))
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


def de_provision():
    url = host + "/v2/service_instances/{}?service_id={}&plan_id={}".format(
        instance_id, service_id, plan_id
    )
    try:
        print("Requesting url {}".format(url))
        r = requests.delete(url, auth=auth, headers=headers, data=json.dumps({
            "service_id": service_id,
            "plan_id": plan_id,
        }))
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


def de_provision_status():
    url = host + "/v2/service_instances/{}/last_operation?operation=deprovision&service_id={}&plan_id={}".format(
        instance_id, service_id, plan_id
    )
    try:
        print("Requesting url {}".format(url))
        r = requests.get(url, auth=auth, headers=headers, data=json.dumps({
            "service_id": service_id,
            "plan_id": plan_id,
        }))
        print("Status {}".format(r.status_code))
        if r.status_code < 400:
            json_body = json.loads(r.content.decode())
            print("Response\n", json.dumps(json_body, indent=2))
    except requests.ConnectionError as e:
        print(e, sys.stderr)
        sys.exit(1)


#catalog()
#provision()
# provision_status()
# bind()
# de_provision()
de_provision_status()
