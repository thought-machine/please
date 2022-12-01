echo $CIRCLE_OIDC_TOKEN > .circleci/CIRCLE_OIDC_TOKEN

gcloud components update
echo login
gcloud auth login --brief --cred-file=.circleci/oicd-provider.json

echo get iam policy
gcloud iam service-accounts get-iam-policy circleci-uploader@tm-please.iam.gserviceaccount.com