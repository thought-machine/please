from behave import given, when, then, step


@given('we have behave installed')
def step_impl(context):
    pass


@given('we should see our custom flags in context')
def step_impl(context):
    assert context.config.userdata['test_value'] == 'this value should show up in my context'


@when('we implement {number:d} tests')
def step_impl(context, number):
    assert number > 1 or number == 0
    context.tests_count = number


@then('behave will test them for us!')
def step_impl(context):
    assert context.failed is False
    assert context.tests_count >= 0


@step('behave finds {number:d} feature files')
def step_impl(context, number):
    assert len(context.config.paths) == 3
