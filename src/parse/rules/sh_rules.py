""" Rules to 'build' shell scripts.

Note that these do pretty much nothing beyond collecting the files. In future we might
implement something more advanced (ala .sar files or whatever).
"""

def sh_library(name, src, deps=None, visibility=None, link=True):
    """Generates a shell script binary, essentially just the given source.

    Note that these are individually executable so can only have one source file each.
    This is a bit tedious and would be nice to improve sometime.

    Args:
      name (str): Name of the rule
      src (str): Source file for the rule
      deps (list): Dependencies of this rule
      visibility (list): Visibility declaration of the rule.
      link (bool): If True, outputs will be linked in plz-out; if False they'll be copied.
    """
    filegroup(
        name=name,
        srcs=[src],
        deps=deps,
        visibility=visibility,
        link=link,
        binary=True,
    )


def sh_binary(name, main, deps=None, visibility=None, link=True):
    """Generates a shell binary, essentially just the shell script itself.

    Args:
      name (str): Name of the rule
      main (str): Main script file.
      deps (list): Dependencies of this rule
      visibility (list): Visibility declaration of the rule.
      link (bool): If True, outputs will be linked in plz-out; if False they'll be copied.
    """
    filegroup(
        name=name,
        srcs=[main],
        deps=deps,
        visibility=visibility,
        link=link,
        binary=True,
    )


def sh_test(name, src=None, args=None, labels=None, data=None, deps=None,
            visibility=None, flaky=0, test_outputs=None, timeout=0, container=False):
    """Generates a shell test. Note that these aren't packaged in a useful way.

    Args:
      name (str): Name of the rule
      src (str): Test script file.
      args (list): Arguments that will be passed to this test when run.
      labels (list): Labels to apply to this test.
      data (list): Runtime data for the test.
      deps (list): Dependencies of this rule
      visibility (list): Visibility declaration of the rule.
      timeout (int): Maximum length of time, in seconds, to allow this test to run for.
      flaky (int | bool): True to mark this as flaky and automatically rerun.
      test_outputs (list): Extra test output files to generate from this test.
      container (bool | dict): True to run this test within a container (eg. Docker).
    """
    build_rule(
        name=name,
        srcs=[src or test],
        data=data,
        deps=deps,
        outs=[name + '.sh'],
        cmd='ln -s ${SRC} ${OUT}',
        test_cmd='$(exe :%s) %s' % (name, ' '.join(args or [])),
        visibility=visibility,
        labels=labels,
        binary=True,
        test=True,
        no_test_output=True,
        flaky=flaky,
        test_outputs=test_outputs,
        test_timeout=timeout,
        container=container,
    )
