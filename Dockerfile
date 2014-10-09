# Ubuntu with python & node
FROM dockerfile/nodejs

MAINTAINER lighthouse


# ----- Install go -----
WORKDIR /root
RUN \
  mkdir -p /goroot && \
  curl https://storage.googleapis.com/golang/go1.3.1.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1

ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

WORKDIR /data

# ----- Add private ssh key -----
RUN mkdir -p /root/.ssh

# Private deploy key for lighthouse, "It's a secret to everybody."
RUN echo "-----BEGIN RSA PRIVATE KEY-----\n\
MIIEowIBAAKCAQEAu5RKR50Jr362XY5/QukNsw2r6rSDQLS5EZxlitXPqprDEC4w\n\
nt9GIC3mbnUa+RvG0Sx2p3ZSLyt8rh4W8Emajkyj77V3Ycoc35L13Z5sda9EUX0z\n\
Y08UkXkK58Kihjs3GgbLH1NaUlmVqI8QHDmIKsJi2Kzp5ufja8pZII4ifVZWWmhu\n\
Yt1JzekxhQNSDY5NDMPhV7NKdz6qMnMgvJtBsORbcharvtgvcqOliP1pleQs5zzi\n\
MYnJeUu+W8HFoqjNLIdxX9CK/weXN8LZLxlE1QsJ2OI1uxEXTthtf1LyB61U2BAR\n\
xpZM4imZR8mIx+SPwNwbMxMhCQz+tIALRYMxRwIDAQABAoIBACQtlaX6Q8P1THb+\n\
5Myi5mGCYYYDCs2QDaG36F2+ny7oanbUccwyg/Pw5mCndWxWTyJI0Rm7WF6ApKtw\n\
Yjw19fk8DuJMvZm+wZLdZU45H/ISu7p7y018Ext7nP7WK0J4aUg7xzFjgigf3x2D\n\
ejf3YKvekfH4Z6SBVPuVK1t8Dmrdx1oeoySThSVUuKd4bbVgpkHRx6R3yx5Og4bg\n\
AJvwBIxwaR0emUH/PRjNziIKSIUX/KyEetZwLThJbwch5JhwJinT8upe5o7/S3w6\n\
HoW/ZP/MuGrGLVp4DG9jndQLqxPEH4r93QAy6+omhZOayOJ/TAj+z1g3ft/LxrLq\n\
e8X3vAECgYEA6ObH9YfbqTSqiIAkf9zQ3LlC620UCdqG0wxGywabt9Z+lVVpvqK1\n\
ldmH2OHjtMrQ6sXIBWGtB35gHACY/EErjfAAsyv/m4eFiPhkCcNF+m/NirGS7VWP\n\
QzywLfWPO93UZjmmrXG5aa1dz5OURWjXn8mzk27UQhe2v9t/FNK1MXECgYEAzi7O\n\
iu/vSJoEXzZtn1vfOnvY6SboLBLNMM1P1CxDlEtkColmemJINU21QklHge4P+2V7\n\
17rmZ7kA35cXDOXuC4Kj+TayIby8fxucfHeIipLlgzbkgWqVIbFD2U6YTnqNXH1p\n\
OuRrOXaHazM6hOJzCwSMfjoPvWr219N4kF0FsjcCgYAOkgGIdstjNoxEpd+isCnQ\n\
5TYujFBonWc55Na49NzhD2Yz6XgIGR3LFiTNiLQ6J0YSqfTtgULV6S4SEmd/wIP9\n\
CTrB+squ7DeKbh+0DKdgF4aAWsOaXXPs/Or4tRgU4rfa/VhUGX1EAziPN+hav0he\n\
ErxNSO22hM1GC3FT2CrFwQKBgBAVJpc/z/Jh0SV8IWDk0azGLE1Dc6i8brT3ztpF\n\
+Z9/ofYQcaXqNKezwAfDn4hLAYQijl5tfbtpet/18R5YcREEx7WQxqRLDIj9pl8v\n\
E797ZduuVHSj064lHZ29u7Oja5NjVOn7F0IMNNPv0wi6gS7C1BKkhMXJqid7n1Pj\n\
baZRAoGBAOBkKYCOEYrMiyVD4t9/m6hQ6avJQZ+tz2SCbQoIu1MrtV8vD2i07uNo\n\
QVVPvdujRMcpI7vA/2gyhhu7RDRhN/yAFeChadfrxFChcWogncfj1DHGNFyjiL64\n\
N2Pt8veiLc0cgXHeGE5rd+zXyeGcmra2W9pfptd49luerYEFQmIC\n\
-----END RSA PRIVATE KEY-----" >> /root/.ssh/id_rsa

RUN chmod 700 /root/.ssh/id_rsa
RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config


# ----- Setup enviroment -----
RUN git clone git@github.com:ngmiller/lighthouse.git
WORKDIR /data/lighthouse

RUN git fetch origin
# Set your development branch here; uncomment for a production deploy
# RUN git checkout <DEV BRANCH NAME HERE>

RUN go get github.com/fsouza/go-dockerclient

# Build front-end
WORKDIR /data/lighthouse/client
RUN npm install
RUN npm install -g gulp
RUN gulp build

# Build/run server
WORKDIR /data/lighthouse/backend/static

EXPOSE 5000
