FROM qmcgaw/godevcontainer:alpine

RUN apk add make curl gpg dirmngr bash sudo bat file

# install my dotfiles
RUN mkdir -p /root/.config && cd /root/.config && git clone https://github.com/cedi/dotfiles.git && cd /root/.config/dotfiles && ./install.sh

RUN export GO111MODULE=on
