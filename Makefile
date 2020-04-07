#!/usr/bin/make -f
SDK_RELEASE_VERSION=v0.16.0

#################################################
# Bootstrapping for K8s Operator SDK
#################################################

ifneq ($(shell which operator-sdk 2> /dev/null),)
bootstrap:
	operator-sdk version
else
bootstrap:
	curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${SDK_RELEASE_VERSION}/operator-sdk-${SDK_RELEASE_VERSION}-x86_64-linux-gnu
	curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${SDK_RELEASE_VERSION}/operator-sdk-${SDK_RELEASE_VERSION}-x86_64-linux-gnu.asc
	chmod +x operator-sdk-${SDK_RELEASE_VERSION}-x86_64-linux-gnu \
		&& sudo mkdir -p /usr/local/bin/ \
		&& sudo cp operator-sdk-${SDK_RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk \
		&& rm operator-sdk-${SDK_RELEASE_VERSION}-x86_64-linux-gnu
	operator-sdk version
endif

generate: bootstrap
	operator-sdk generate k8s