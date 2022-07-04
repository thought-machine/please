TOOLS_VERSION="1.0.0"

remote_file(
    name = "please_pex",
    url = f"https://github.com/please-build/python-rules/releases/download/tools-v{TOOLS_VERSION}/please_pex-{TOOLS_VERSION}-{CONFIG.HOSTOS}_{CONFIG.HOSTARCH}",
    hashes = [
        "dd25e223262b7a2883709d690d2fde7163fea6ad6eed459e71a0187b8cdc372b", # linux_amd64
        "f8597d85346dd2350cd8bf583f7d6eef3690a28c9324d6b1323eef8d70e50c90", # linux_arm64
        "2b65e61b36c7b0771cabd50b0dc1f811f6f167e1c68cb9e56876f02ec32cce93", # darwin_amd64
        "6b0e3196c2812686c739e474b05cc6498fcf552aa85f5dcd871af890205f83be", # darwin_arm64
        "f0659c434d9a8e9e42cb6f0706f6613c3a3e6ed6c321f67a3db4d7fc8e238707", # freebsd_amd64
    ],
    binary = True,
    visibility = ["PUBLIC"],
)
