import Collapsible from "./Collapsible";

interface ProvidedProps {
  defaultCollapsed?: boolean;
}

const Explanation = ({ defaultCollapsed = false }: ProvidedProps) => {
  return (
    <div className="grid grid-cols-3 gap-x-4">
      <Collapsible
        title="🎯 Why did we build this?"
        defaultCollapsed={defaultCollapsed}
      >
        <p>
          This benchmark tool was built to help us understand and optimize the
          performance of our blockchain infrastructure. Specifically, it helps
          us:
        </p>
        <ul>
          <li>
            <span className="font-medium">Measure real-world performance:</span>{" "}
            By simulating actual transaction loads and block production, we can
            identify bottlenecks and performance issues in our system.
          </li>
          <li>
            <span className="font-medium">Track improvements:</span> As we make
            changes to our infrastructure, we can measure their impact on
            performance and ensure we&apos;re moving in the right direction.
          </li>
          <li>
            <span className="font-medium">Compare configurations:</span> We can
            test different network configurations, hardware setups, and software
            versions to find the optimal setup for our needs.
          </li>
        </ul>
      </Collapsible>

      <Collapsible
        title="💡 What do these numbers mean?"
        defaultCollapsed={defaultCollapsed}
      >
        <p>Each benchmark run consists of a few stages:</p>
        <ol>
          <li>
            <span className="font-medium">
              📥 Submit transactions to the mempool:
            </span>{" "}
            Transactions are submitted to the mempool and validated by the
            sequencer. This calls <code>eth_sendRawTransaction</code> on the
            sequencer. The latency to send all transactions in each block is
            tracked by the <code>send_txs</code> column.
          </li>
          <li>
            <span className="font-medium">🧱 Build and seal blocks:</span> The
            sequencer builds blocks using transactions from the mempool,
            computes the state root, and seals the block. This calls{" "}
            <code>forkChoiceUpdated</code> (start block building) and{" "}
            <code>getPayload</code> (fetch built block) repeatedly.
          </li>
          <li>
            <span className="font-medium">
              👀 Validator nodes receive sealed blocks:
            </span>{" "}
            Validator nodes receive sealed blocks from the sequencer and verify
            them. When a block is received, the validator will call{" "}
            <code>newPayload</code> to verify the block.
          </li>
        </ol>
      </Collapsible>

      <Collapsible
        title="🤝 How can I contribute?"
        defaultCollapsed={defaultCollapsed}
      >
        <p>
          We welcome contributions from the community! Here are some ways you
          can help:
        </p>
        <ul>
          <li>
            <span className="font-medium">Run benchmarks:</span> Test the tool
            with different configurations and share your results. This helps us
            understand performance across various environments.
          </li>
          <li>
            <span className="font-medium">Report issues:</span> If you encounter
            any bugs or have suggestions for improvements, please open an issue
            on our GitHub repository.
          </li>
          <li>
            <span className="font-medium">Submit PRs:</span> We welcome pull
            requests for bug fixes, new features, or documentation improvements.
            Make sure to follow our contribution guidelines.
          </li>
          <li>
            <span className="font-medium">Share feedback:</span> Let us know how
            you&apos;re using the tool and what additional features would be
            helpful for your use case.
          </li>
        </ul>
        <p>
          The code for this repo is available{" "}
          <a href="//github.com/base/benchmark">here</a>.
        </p>
      </Collapsible>
    </div>
  );
};

export default Explanation;
