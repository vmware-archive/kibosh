import unittest

from test_bind import *
from test_catalog import *
from test_provision import *


def suite():
    s = unittest.TestSuite()

    instance_id = uuid.uuid4()
    # instance_id = "41ef848a-3486-4316-8912-1fbc290510f9"

    TestCatalog.instance_id = instance_id
    s.addTest(TestCatalog('test_catalog'))

    if False:
        test_provision = TestProvision('test_provision')
        test_provision.instance_id = instance_id
        s.addTest(test_provision)

    TestBindUnbind.instance_id = instance_id
    # todo: manually adding test **will** bite us. Evaulate more.
    s.addTests([
        TestBindUnbind('test_bind_response_credentials'),
        TestBindUnbind('test_bind_template'),
        TestBindUnbind('test_connection'),
        TestBindUnbind('test_unbind_response')
    ])

    return s


if __name__ == '__main__':
    runner = unittest.TextTestRunner()
    runner.run(suite())
