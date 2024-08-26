prep-python::
	./demo/python/init.sh

sa-key::
	yc iam service-account create --name metering || echo "Service account created";
	export SA_ID=`yc iam service-account get --name metering --format json | jq -r .id`; \
	export FOLDER_ID=`yc config get folder-id`; \
	yc resource-manager folder add-access-binding --role marketplace.meteringAgent --subject serviceAccount:$$SA_ID --id $$FOLDER_ID; \
	yc iam key create --service-account-id $$SA_ID --output key.json --description "Key for metering service account"
