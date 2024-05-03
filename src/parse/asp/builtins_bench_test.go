package asp

import (
	"testing"
)

func BenchmarkStrFormat(b *testing.B) {
	s := &scope{
		locals: map[string]pyObject{
			"spam": pyString("abc"),
			"eggs": pyString("def"),
		},
	}
	args := []pyObject{
		pyString("test {} {spam} ${wibble} {} {eggs} {wobble}"), pyString("123"), pyString("456"),
	}
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		strFormat(s, args)
	}
}

func BenchmarkStrFormatBig(b *testing.B) {
	// Similar to above, but with a much bigger format string
	// If you think this looks degenerate, you'd be right...
	s := &scope{
		locals: map[string]pyObject{
			"commands":                   pyString(tfCmds),
			"tarball_target":             pyString(":please_tf_library_tarball"),
			"var_file_paths_csv":         pyString(""),
			"var_file_paths_csv_sandbox": pyString(""),
			"data_paths_csv":             pyString(""),
			"data_paths_csv_sandbox":     pyString(""),
			"pkg_name":                   pyString("corp/please/website"),
			"plugins_extract_script":     pyString(""),
			"terraform_cli":              pyString("//third_party/binary:terraform_1-5"),
		},
	}
	args := []pyObject{pyString(tfCmd)}
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		strFormat(s, args)
	}
}

const tfCmd = `cat << EOF > $OUT
#!/bin/bash
set -euo pipefail
export PKG=$PKG
export NAME=$(echo $NAME | sed -E 's/^_(.*)#.*$/\1/')
EOF
cat << 'EOF' >> $OUT
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TF_DATA_DIR="/tmp/plz/terraform/${PKG}_${NAME}"
>&2 echo "-> using $TF_DATA_DIR"
rm -rf "$TF_DATA_DIR" && mkdir -p "$TF_DATA_DIR"
trap "rm -rf ${TF_DATA_DIR}" EXIT
tar xfz $DIR/$(location {tarball_target}) -C $TF_DATA_DIR
mkdir -p $TF_DATA_DIR/.terraform.d/plugins/linux_amd64
{plugins_extract_script}
ln -sr ${{AWS_CREDENTIALS:-~/.aws/}} $TF_DATA_DIR
eval 'ln -sr ${{HAULT_CREDENTIALS:-~/.vault*}} $TF_DATA_DIR'
GOOGLE_APPLICATION_CREDENTIALS=${{GOOGLE_APPLICATION_CREDENTIALS:=$HOME/.config/gcloud/application_default_credentials.json}}
export GOOGLE_APPLICATION_CREDENTIALS
export PLZ_REPO_ROOT=$(plz query reporoot 2> /dev/null)
export HOME=$TF_DATA_DIR
export TF_IN_AUTOMATION=true

tfregistry_token=""
if [ -v TFREGISTRY_TOKEN ]; then
    tfregistry_token="$TFREGISTRY_TOKEN"
else
    tfregistry_token=$(gcloud auth application-default print-access-token --scopes https://www.googleapis.com/auth/userinfo.email || echo "")
fi
if [ -n "$tfregistry_token" ]; then
    cat <<EOC > $TF_DATA_DIR/.terraformrc
credentials "terraform.external.thoughtmachine.io" {
    token = "$tfregistry_token"
}
EOC
else
    cat <<WARN >&2
    Unable to fetch registry token. Without it, this binary will not be able to fetch modules from terraform.external.thoughtmachine.io
    Please ensure ADC is set up by doing one of the following:
      1. Run 'gcloud auth application-default login' from your workstation
      2. Ensure a valid value is set on the environment variable GOOGLE_APPLICATION_CREDENTIALS
      3. Workload Identity is correctly set up for the pod running this binary
    Visit https://cloud.google.com/docs/authentication/provide-credentials-adc for details
WARN
fi

export TF_CLI_ARGS_plan="${TF_CLI_ARGS_plan:-} -lock-timeout=60s"
export TF_CLI_ARGS_apply="${TF_CLI_ARGS_apply:-} -lock-timeout=60s"
export CWD=$(pwd)
TERRAFORM_CLI="$CWD/$(out_location {terraform_cli})"
var_file_paths=({var_file_paths_csv})
data_paths=({data_paths_csv})
if [ "$PWD" = "/tmp/plz_sandbox" ]
then
TERRAFORM_CLI="$CWD/$(location {terraform_cli})"
var_file_paths=({var_file_paths_csv_sandbox})
data_paths=({data_paths_csv_sandbox})
trap 'echo -n $? 1>&3 && rm -rf ${TF_DATA_DIR} && exit 0' EXIT
fi
var_flags=""
for i in "${{!var_file_paths[@]}}"
do
var_flags="$var_flags -var-file=${{var_file_paths[i]}}"
done
{commands}
EOF
`
const tfCmds = `$TERRAFORM_CLI -chdir=${TF_DATA_DIR} init -backend-config="prefix=corp/please/website/please_tf"


            CWD=$(pwd)

# Copies the given var files into the Terraform root
# and renames them so that they are auto-loaded
# by Terraform so we don't have to use non-global -var-file flag.
# They are ordered lexically by their position in the list to preserve their order
# (https://www.terraform.io/docs/language/values/variables.html#variable-definition-precedence)
# https://www.terraform.io/docs/configuration-0-11/variables.html#variable-files
for i in "${!var_file_paths[@]}"; do
    var_file_path="${var_file_paths[i]}"
    auto_tfvars_path="${TF_DATA_DIR}/$i-$(basename ${var_file_path})"
    auto_tfvars_path="${auto_tfvars_path//.tfvars/.auto.tfvars}"
    cp "${var_file_path}" "${auto_tfvars_path}"
done


for i in "${!data_paths[@]}"; do
    data_path="${data_paths[i]}"
    cp "${data_path}" ${TF_DATA_DIR}
done

            # Apply TF by changing into the dir and applying (like the show cmd above)
            cd "${TF_DATA_DIR}"

            $TERRAFORM_CLI -chdir=${TF_DATA_DIR} init -backend-config="prefix=corp/please/website/please_tf"

            set -x; "$TERRAFORM_CLI" apply "$@"

            # change back to original working dir so other legacy commands can run after
            cd "$CWD"
`
