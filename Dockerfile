# syntax=docker/dockerfile:1.2

# TODO: Use pre-built images once they are available to we avoid all this cloning/building
FROM rust:1.61-buster as aptos-builder
ENV CARGO_NET_GIT_FETCH_WITH_CLI=true
RUN apt-get update && apt-get install -y cmake curl clang git pkg-config libssl-dev libpq-dev
RUN --mount=type=cache,target=/var/cache/apk \
    --mount=type=cache,target=/home/rust/.cargo \
    rustup component add rustfmt \
    # Use branch add_sf_stream_thread until it's merged upstream and in which case it will not be required anymore
    && git clone https://github.com/aptos-labs/aptos-core.git -b add_sf_stream_thread \
    && cd aptos-core \
    # In `debug` mode for now just to speed up compilation because I don't want to wait too long for it
    && RUSTFLAGS="--cfg tokio_unstable" cargo build -p aptos-node \
    && cp target/debug/aptos-node /home/rust/

FROM ubuntu:20.04

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get -y install -y \
    ca-certificates libssl1.1 vim htop iotop sysstat \
    dstat strace lsof curl jq tzdata && \
    rm -rf /var/cache/apt /var/lib/apt/lists/*

RUN rm /etc/localtime && ln -snf /usr/share/zoneinfo/America/Montreal /etc/localtime && dpkg-reconfigure -f noninteractive tzdata

RUN mkdir /tmp/wasmer-install && cd /tmp/wasmer-install && \
    curl -L https://github.com/wasmerio/wasmer/releases/download/2.3.0/wasmer-linux-amd64.tar.gz | tar xzf - && \
    mv lib/libwasmer.a lib/libwasmer.so /usr/lib/ && cd / && rm -rf /tmp/wasmer-install

ADD /fireaptos /app/fireaptos
COPY --from=aptos-builder /home/rust/aptos-node /app/aptos-node

COPY tools/fireaptos/motd_generic /etc/
COPY tools/fireaptos/motd_node_manager /etc/
COPY tools/fireaptos/99-firehose.sh /etc/profile.d/
COPY tools/fireaptos/scripts/* /usr/local/bin

ENTRYPOINT ["/app/fireaptos"]
