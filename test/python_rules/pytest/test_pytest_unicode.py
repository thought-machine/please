def test_unicode_latin():
    assert 'kérem' == 'kérem'


def test_unicode_emoji():
    assert '✂' == '✂'
    assert '✔' != '❄'
