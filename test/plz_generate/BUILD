subinclude("//test/build_defs")

do_not_edit_warning = "# Entries below this point are managed by Please (DO NOT EDIT)"

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
        ".gitignore": "\n".join(user_entries) + f"\n\n{do_not_edit_warning}\n" + "\n".join(all_generated_files),
        "src/foo.gen": "wibble",
        "src/bar/bar.gen": "wobble",
    },
    plz_command = f"{init_cmd} && plz generate --update_gitignore .gitignore",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "generate_update_gitignore_respects_directories_test",
    expected_output = {
        ".gitignore": "\n".join([
            "**/bar/*.gen",
            "/baz.gen",
            f"\n{do_not_edit_warning}",
            "src/foo.gen",
            # src/bar/bar.gen not added, because it is covered by **/bar/*.gen
            "src/bar/baz/baz.gen",  # Added, because it is not covered by /baz.gen
        ]),
    },
    plz_command = "git init && echo '**/bar/*.gen\n/baz.gen' > .gitignore && plz generate --update_gitignore .gitignore",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "generate_bar_test",
    expected_output = {
        "src/bar/.gitignore": f"\n{do_not_edit_warning}\n" + "\n".join(bar_generated_files),
        "src/bar/bar.gen": "wobble",
    },
    plz_command = f"git init && plz generate --update_gitignore src/bar/.gitignore",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "auto_update_generate_bar_test",
    expected_output = {
        "src/bar/baz/.gitignore": f"\n{do_not_edit_warning}\nbaz.gen",
    },
    plz_command = f"git init && plz build --profile auto //src/bar/baz",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "auto_update_generate_glob_test",
    expected_output = {
        ".gitignore": f"\n{do_not_edit_warning}\nsrc/bar/baz/*.gen",
        "src/bar/.gitignore": f"\n{do_not_edit_warning}\nbar.gen",
        "src/.gitignore": f"\n{do_not_edit_warning}\nfoo.gen",
    },
    plz_command = f"git init && echo '\n{do_not_edit_warning}\nsrc/bar/baz/*.gen' > .gitignore && plz build --profile auto",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "auto_update_generate_glob_all_gen_test",
    expected_output = {
        ".gitignore": f"\n{do_not_edit_warning}\n*.gen",
        "src/bar/.gitignore": "",
        "src/.gitignore": "",
    },
    plz_command = f"git init && echo '\n{do_not_edit_warning}\n*.gen' > .gitignore && touch src/bar/.gitignore && touch src/.gitignore && plz build --profile auto",
    repo = "test_repo",
)
