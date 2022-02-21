FROM registry.access.redhat.com/ubi8/go-toolset:1.16.7 as builder
ENV GOPATH=$APP_ROOT
COPY --chown=1001:0 . .
RUN make cmd

FROM registry.access.redhat.com/ubi8/openjdk-8-runtime
USER root
RUN echo -e "[centos8]" \
 "\nname = centos8" \
 "\nbaseurl = http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/" \
 "\nenabled = 1" \
 "\ngpgcheck = 0" > /etc/yum.repos.d/centos.repo
RUN microdnf -y install wget git subversion \
 && microdnf -y clean all
RUN wget -qO /opt/windup.zip https://repo1.maven.org/maven2/org/jboss/windup/mta-cli/5.2.1.Final/mta-cli-5.2.1.Final-offline.zip \
 && unzip /opt/windup.zip -d /opt \
 && rm /opt/windup.zip \
 && ln -s /opt/mta-cli-5.2.1.Final/bin/mta-cli /opt/windup  
COPY --from=builder /opt/app-root/src/bin/addon /usr/local/bin/addon
USER jboss
ENTRYPOINT ["/usr/local/bin/addon"]
