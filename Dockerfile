FROM golang:1.20

# 開発用
RUN go install github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest
RUN go install github.com/ramya-rao-a/go-outline@latest
RUN go install github.com/cweill/gotests/gotests@latest
RUN go install github.com/fatih/gomodifytags@latest
RUN go install github.com/josharian/impl@latest
RUN go install github.com/haya14busa/goplay/cmd/goplay@latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install honnef.co/go/tools/cmd/staticcheck@latest
RUN go install golang.org/x/tools/gopls@latest
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# ツール
RUN go install github.com/goark/depm@latest
RUN go install github.com/magefile/mage@latest
RUN curl https://gotest-release.s3.amazonaws.com/gotest_linux > /usr/local/bin/gotest && chmod +x /usr/local/bin/gotest

RUN go install github.com/cosmtrek/air@latest

COPY . /opt/app
WORKDIR /opt/app

CMD ./main_app