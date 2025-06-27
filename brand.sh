#!/bin/bash

brand="$1"
app_version="$2"

cert_dir="self-certification"

guance_brand_domain="guance.com"
guance_brand_repo_datakit_operator="pubrepo.guance.com/datakit-operator"
guance_brand_repo_dataflux="pubrepo.guance.com/dataflux"
guance_brand_repo_datakit="pubrepo.guance.com/datakit"
guance_repository="$guance_brand_repo_datakit_operator/datakit-operator"

truewatch_brand_domain="truewatch.com"
truewatch_brand_repo_datakit_operator="pubrepo.truewatch.com/truewatch"
truewatch_brand_repo_dataflux="pubrepo.truewatch.com/truewatch"
truewatch_brand_repo_datakit="pubrepo.truewatch.com/truewatch"
truewatch_repository="$truewatch_brand_repo_datakit_operator/datakit-operator"

testing_brand_domain="guance.com"
testing_brand_repo_datakit_operator="pubrepo.guance.com/datakit-operator"
testing_brand_repo_dataflux="pubrepo.guance.com/dataflux"
testing_brand_repo_datakit="pubrepo.guance.com/datakit"
testing_repository="registry.jiagouyun.com/datakit-operator"


# check input parameters
if [ -z "$brand" ]; then
    echo "Error: invalid brand parameter, exit"
    exit 1
elif [ "$brand" != "guance" ] && [ "$brand" != "truewatch" ] && [ "$brand" != "testing" ]; then
    echo "Error: brand must be either 'guance' or 'truewatch' or 'testing'"
    exit 1
elif [ -z "$app_version" ]; then
    echo "Error: invalid app_version parameter, exit"
    exit 1
fi


if [ "$brand" = "guance" ]; then
	sed -e "s,(@BRAND_DOMAIN),$guance_brand_domain,g" \
		templates/charts-README.template.md > charts/datakit-operator/README.md

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@BRAND_DOMAIN),$guance_brand_domain,g" \
		templates/charts-Chart.template.yaml > charts/datakit-operator/Chart.yaml

	sed -e "s,(@REPOSITORY),$guance_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$guance_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$guance_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$guance_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/charts-values.template.yaml > charts/datakit-operator/values.yaml

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@REPOSITORY),$guance_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$guance_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$guance_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$guance_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/datakit-operator.template.yaml > datakit-operator.yaml

elif [ "$brand" = "truewatch" ]; then
	sed -e "s,(@BRAND_DOMAIN),$truewatch_brand_domain,g" \
		templates/charts-README.template.md > charts/datakit-operator/README.md

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@BRAND_DOMAIN),$truewatch_brand_domain,g" \
		templates/charts-Chart.template.yaml > charts/datakit-operator/Chart.yaml

	sed -e "s,(@REPOSITORY),$truewatch_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$truewatch_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$truewatch_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$truewatch_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/charts-values.template.yaml > charts/datakit-operator/values.yaml

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@REPOSITORY),$truewatch_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$truewatch_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$truewatch_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$truewatch_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/datakit-operator.template.yaml > datakit-operator.yaml

elif [ "$brand" = "testing" ]; then
	sed -e "s,(@BRAND_DOMAIN),$testing_brand_domain,g" \
		templates/charts-README.template.md > charts/datakit-operator/README.md

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@BRAND_DOMAIN),$testing_brand_domain,g" \
		templates/charts-Chart.template.yaml > charts/datakit-operator/Chart.yaml

	sed -e "s,(@REPOSITORY),$testing_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$testing_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$testing_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$testing_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/charts-values.template.yaml > charts/datakit-operator/values.yaml

	sed -e "s,(@APP_VERSION),$app_version,g" \
		-e "s,(@REPOSITORY),$testing_repository,g" \
		-e "s,(@BRAND_REPO_DATAKIT_OPERATOR),$testing_brand_repo_datakit_operator,g" \
		-e "s,(@BRAND_REPO_DATAFLUX),$testing_brand_repo_dataflux,g" \
		-e "s,(@BRAND_REPO_DATAKIT),$testing_brand_repo_datakit,g" \
	       	-e "s/(@CABUNDLE)/`cat $cert_dir/tls.crt | base64 | tr -d "\n"`/g" \
		templates/datakit-operator.template.yaml > datakit-operator.yaml
else
    echo "Fatal：unreachable!"
    exit 1
fi

