subinclude("//test/build_defs")

do_not_edit_warning = "# Entries below this point are managed by Please (DO NOT EDIT)"

default_ignored_files = [
    "plz-out",
    ".plzconfig.local",
]

user_entries = [
    ".idea",
    "*.iml",
]

all_generated_files = [
    "src/foo.gen",
    "src/bar/bar.gen",
    "src/bar/baz/baz.gen",
]

bar_generated_files = [
    "bar.gen",
    "baz/baz.gen",
]

init_cmd = "git init && " + " && ".join([f"echo {entry} >> .gitignore" for entry in user_entries])

please_repo_e2e_test(
    name = "generate_all_test",
    expected_output = {
        ".gitignore": "\n".join(user_entries) + f"\n\n{do_not_edit_warning}\n" + "\n".join(default_ignored_files + all_generated_files),
        "src/foo.gen": "wibble",
        "src/bar/bar.gen": "wobble",
    },
    plz_command = f"{init_cmd} && plz generate --update_gitignore .gitignore",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "generate_bar_test",
    expected_output = {
        "src/bar/.gitignore": f"{do_not_edit_warning}\n" + "\n".join(bar_generated_files),
        "src/bar/bar.gen": "wobble",
    },
    plz_command = f"git init && plz generate --update_gitignore src/bar/.gitignore",
    repo = "test_repo",
)
