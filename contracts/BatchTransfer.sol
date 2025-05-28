// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
contract BatchTransfer {
    address public owner;
    constructor() {
        owner = msg.sender; // 部署合约的人为owner
    }
    modifier onlyOwner() {
        require(msg.sender == owner, "Not the owner");
        _;
    }
    // 当直接向合约发送 ETH 时触发 (>=0.6.0版本使用receive)
    receive() external payable {
        // 合约可接收ETH，不执行额外逻辑
    }
    // 可选：fallback函数，在未匹配函数或调用时也可接收ETH
    fallback() external payable { }
    // 批量发送不等额金额给多个地址
    function batchSend(address[] calldata recipients, uint256[] calldata amounts) external payable {
        require(recipients.length == amounts.length, "Length mismatch");
        uint256 totalAmount = 0;
        for (uint i = 0; i < amounts.length; i++) {
            totalAmount += amounts[i];
        }
        require(msg.value >= totalAmount, "Not enough ETH sent");
        for (uint i = 0; i < recipients.length; i++) {
            (bool success,) = payable(recipients[i]).call{value: amounts[i]}("");
            require(success, "Transfer failed");
        }
    }
    // 批量发送相同金额给多个地址
    function batchSendEqual(address[] calldata recipients, uint256 amountEach) external payable {
        uint256 count = recipients.length;
        uint256 totalAmount = amountEach * count;
        require(msg.value >= totalAmount, "Not enough ETH sent");
        for (uint i = 0; i < count; i++) {
            (bool success,) = payable(recipients[i]).call{value: amountEach}("");
            require(success, "Transfer failed");
        }
    }
    // Owner将合约中的全部余额转出到指定地址
    function withdraw(address payable to) external onlyOwner {
        uint256 balance = address(this).balance;
        require(balance > 0, "No ETH to withdraw");
        (bool success, ) = to.call{value: balance}("");
        require(success, "Withdraw failed");
    }
    // 查询合约余额
    function getBalance() external view returns (uint256) {
        return address(this).balance;
    }
}