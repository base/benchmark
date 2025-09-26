# Stage 1: Base with system dependencies, Rust, and source code
FROM ubuntu:22.04 AS base

ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/root/.cargo/bin:${PATH}"

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    git \
    make \
    cmake \
    pkg-config \
    libssl-dev \
    libpq-dev \
    libclang-dev \
    ca-certificates \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Install Rust
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y

# Install Go
# Install Go manually using curl
RUN curl -fsSL https://go.dev/dl/go1.23.0.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:$PATH"

# Clone all source repos
WORKDIR /opt
RUN git clone https://github.com/paradigmxyz/reth.git && \
    git clone https://github.com/ethereum-optimism/op-geth.git && \
    git clone https://github.com/base/benchmark.git

RUN git -C reth checkout --force 9d56da53ec0ad60e229456a0c70b338501d923a5 && \
    git -C op-geth checkout --force 4bc345b22fbee14d3162becd197373a9565b7c6d

# Stage 2: Build reth
FROM base AS build-reth
WORKDIR /opt/reth
RUN cargo build --features asm-keccak --profile release --bin op-reth --manifest-path crates/optimism/bin/Cargo.toml

# Stage 3: Build geth
FROM base AS build-geth
WORKDIR /opt/op-geth
RUN make geth

# Stage 4: Build benchmark
FROM base AS build-benchmark
WORKDIR /opt/benchmark
RUN make build

# Stage 5: Final minimal runtime image
FROM ubuntu:22.04 AS runtime

# Install minimal runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy binaries from build stages
COPY --from=build-benchmark /opt/benchmark /opt/benchmark
COPY --from=build-benchmark /opt/benchmark/bin/base-bench /usr/local/bin/base-bench
COPY --from=build-reth /opt/reth/target/release/op-reth /usr/local/bin/reth
COPY --from=build-geth /opt/op-geth/build/bin/geth /usr/local/bin/geth

# Set working directory and default command
WORKDIR /opt/benchmark
ENTRYPOINT ["base-bench"]
