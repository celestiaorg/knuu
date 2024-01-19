# ADR 001: Redesigning Knuu for Asynchronous Testing Approach

## Status

Draft

## Context

The existing design of `Knuu` requires exposing the addresses and ports of the target instance running alongside BitTwister, complicating the configuration process. This setup requires Knuu to orchestrate the instances, configure them, and execute the test, leading to complexities in managing traffic shaping. Moreover, as the current model relies on port forwarding, scalability issues have surfaced, causing inconsistent errors and hindering seamless test execution.

## Problem

The current synchronous model necessitates exposing the target instance's address and ports to BitTwister via a sidecar. This centralized approach introduces dependencies and complexities in configuring the test environment. Additionally, the reliance on a proxy approach or port forwarding poses scalability issues, resulting in inconsistent errors during test execution.

## Proposed Solution

### A Distributed Approach with a Playbook Model

Embrace a distributed playbook model similar to Testground, allowing asynchronous orchestration and execution of test scenarios:

1. **Test Scenario Definition:** Users define test scenarios in a descriptive language such as [Cue](https://cuelang.org/), which provides more dynamism compared to HCL.
2. **Knuu Instance Initiation:** Knuu initiates instances and assigns specific playbooks to each instance.
3. **BitTwister Sidecar with Playbook Understanding:** BitTwister runs alongside instances with added functionality to interpret and execute the playbook's instructions.
4. **Dynamic Traffic Shaping:** BitTwister adjusts traffic shaping based on predefined events or triggers from the target instance, eliminating the need for continuous centralized control.

##### CI/CD Integration

To enable the execution of tests as cron jobs and verification pipelines for upcoming releases, a CI/CD gateway should be implemented. This gateway will facilitate the integration of Knuu with the CI/CD system, allowing contributors to define and automate test scenarios as part of their release workflows.

Contributors can easily trigger tests, monitor their execution, and receive feedback on the test results. This ensures that the artifacts produced by contributors are thoroughly tested before being released.

### Advantages

1. **Decentralized Execution:** Instances operate autonomously based on pre-defined playbooks, reducing dependency on centralized control.
2. **Elimination of IP Exposure:** Removes the necessity to expose IP addresses or utilize proxies for controlling data.

### Comments

Further discussions revealed the necessity for test designers to predefine scenarios and program them before runtime, a feature lacking in the current synchronous design of Knuu. Moving towards an asynchronous-only approach aligns with the distributed design of Testground and simplifies usage.

### Proposed Roadmap

- **v0:** Consider the current version as complete, encapsulating UX issues with helper functions where needed.
- **v1:** Create a separate branch to implement a new ADR-driven design. This version will gather consensus on the foundational aspects of the new design before delving into implementation details.

### Considerations

- The current reliance on port forwarding poses inconsistent issues and challenges, necessitating a shift towards an encapsulated test environment.
- Use of a Scaleway instance is suggested for scenarios where port forwarding errors persist, enabling smoother execution.

## References

- Discussions with the team on the necessity for an asynchronous approach.
- Insights from @Bidon15, @smuu and @mojtaba-esk regarding the importance of pre-programmed test scenarios.
- Considerations from team members highlighting the challenges with port forwarding and the need for an encapsulated test environment.
