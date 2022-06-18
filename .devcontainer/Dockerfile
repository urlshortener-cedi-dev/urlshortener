FROM qmcgaw/godevcontainer:alpine

RUN apk add make curl gpg dirmngr bash sudo bat file

# install my dotfiles
RUN mkdir -p /root/.config && cd /root/.config && git clone https://github.com/cedi/dotfiles.git && cd /root/.config/dotfiles && ./install.sh

RUN export GO111MODULE=on

# Install Operator SDK
RUN export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)    && \
    export OS=$(uname | awk '{print tolower($0)}')  && \
    export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.22.0 && \
    curl -LO "${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}" && \
    chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
