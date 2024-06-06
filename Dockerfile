FROM registry.access.redhat.com/ubi9/go-toolset:latest as builder
ENV GOPATH=$APP_ROOT
COPY --chown=1001:0 . .
RUN make cmd

FROM registry.access.redhat.com/ubi9/ubi-minimal
USER root
RUN microdnf -y update && microdnf -y clean all
RUN echo -e "[centos9]" \
 "\nname = centos9" \
 "\nbaseurl = http://mirror.stream.centos.org/9-stream/AppStream/\$basearch/os/" \
 "\nenabled = 1" \
 "\ngpgcheck = 0" > /etc/yum.repos.d/centos.repo
RUN microdnf -y install \
java-11-openjdk-headless \
openssh-clients \
unzip \
wget \
git \
subversion \
maven \
&& microdnf -y clean all
ARG WINDUP=https://repo1.maven.org/maven2/org/jboss/windup/tackle-cli/6.3.8.Final/tackle-cli-6.3.8.Final-offline.zip
RUN wget -qO /opt/windup.zip $WINDUP \
 && unzip /opt/windup.zip -d /opt \
 && rm /opt/windup.zip \
 && ln -s /opt/tackle-cli-*/bin/windup-cli /opt/windup

ENV HOME=/working \
    JAVA_HOME="/usr/lib/jvm/jre-11" \
    JAVA_VENDOR="openjdk" \
    JAVA_VERSION="11"
WORKDIR /working
COPY --from=builder /opt/app-root/src/bin/addon /usr/local/bin/addon
ENTRYPOINT ["/usr/local/bin/addon"]
