FROM google/golang

MAINTAINER lighthouse

WORKDIR /gopath/src/github.com/ngmiller/lighthouse

RUN go get github.com/gorilla/mux \
           github.com/gorilla/sessions \
           github.com/gorilla/securecookie \
           github.com/bmizerany/pq \
           code.google.com/p/goauth2/oauth \
           code.google.com/p/google-api-go-client/compute/v1

ADD . /gopath/src/github.com/ngmiller/lighthouse/
RUN go get github.com/ngmiller/lighthouse

ENTRYPOINT ["/gopath/bin/lighthouse"]
