import os
import sys
import unittest
import uuid

from test_bind import *
from test_catalog import *
from test_deprovision import *
from test_provision import *


def suite():
    s = unittest.TestSuite()

    instance_id = uuid.uuid4()

    s.addTest(TestCatalog('test_catalog'))

    TestProvision.instance_id = instance_id
    s.addTest(TestProvision('test_provision'))

    TestBindUnbind.instance_id = instance_id
    s.addTests([
        TestBindUnbind('test_bind_response_credentials'),
        TestBindUnbind('test_bind_template'),
        TestBindUnbind('test_connection'),
        TestBindUnbind('test_unbind_response')
    ])

    TestDeprovision.instance_id = instance_id
    s.addTest(TestDeprovision('test_deprovision'))

    return s


if __name__ == '__main__':
    discovered_tests = unittest.defaultTestLoader.discover(os.path.dirname(os.path.abspath(__file__)))
    discovered_tests.countTestCases()

    s = suite()
    runner = unittest.TextTestRunner(verbosity=2, failfast=True)
    runner.run(s)

    if s.countTestCases() != discovered_tests.countTestCases():
        print(
            "Number of test cases [{}] do not match discovered test cases [{}].".format(
                s.countTestCases(), discovered_tests.countTestCases(),
            ), file=sys.stderr
        )
        print("\tBe sure to add all tests explicitly to the suite in order", file=sys.stderr)
        os.sys.exit(1)
