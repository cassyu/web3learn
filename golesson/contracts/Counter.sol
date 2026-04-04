// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title 作业用简单计数器（任务2：abigen + Sepolia 交互）
contract Counter {
    uint256 public count;

    function increment() external {
        count += 1;
    }
}
