package(
    default_python_interpreter = 'python3',
    pip_tool = 'pip3',
    python_wheel_repo = 'https://get.please.build/third_party/python/py3',
)

python_wheel(
    name = 'coverage',
    version = '4.3.4',
    visibility = ['PUBLIC'],
)

python_wheel(
    name = 'cffi',
    version = '1.10.0',
    outs = [
        '_cffi_backend.so',
        'cffi',
    ],
    post_install_commands = ['mv *.so _cffi_backend.so'],
    visibility = ['PUBLIC'],
)
