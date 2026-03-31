---
title: ERC20
icon: /ERC20.svg
order: 1
---

什么是代币？

代币可以在以太坊中表示任何东西：

* 在线平台中的信誉积分
* 游戏中一个角色的技能
* 金融资产类似于公司股份的资产
* 像美元一样的法定货币
* 一盎司黄金
* 以及更多...

ERC-20 提出了一个同质化代币的标准，换句话说，它们具有一种属性，使得每个代币都与另一个代币（在类型和价值上）完全相同。它实现了代币转账的基本逻辑：

* 账户余额(balanceOf())
* 转账(transfer())
* 授权转账(transferFrom())
* 授权(approve())
* 代币总供给(totalSupply())
* 授权转账额度(allowance())
* 代币信息（可选）：名称(name())，代号(symbol())，小数位数(decimals())

`IERC20`是`ERC20`代币标准的接口合约，规定了`ERC20`代币需要实现的函数和事件。 

`IERC20`定义了`2`个事件：`Transfer`事件和`Approval`事件，分别在转账和授权时被释放。

```solidity
/**
 * @dev 释放条件：当 `value` 单位的货币从账户 (`from`) 转账到另一账户 (`to`)时.
 */
event Transfer(address indexed from, address indexed to, uint256 value);

/**
 * @dev 释放条件：当 `value` 单位的货币从账户 (`owner`) 授权给另一账户 (`spender`)时.
 */
event Approval(address indexed owner, address indexed spender, uint256 value);
```



`IERC20`定义了`6`个函数，提供了转移代币的基本功能，并允许代币获得批准，以便其他链上第三方使用。

```solidity
/**
 * @dev 返回代币总供给.
 */
function totalSupply() external view returns (uint256);

/**
 * @dev 返回账户`account`所持有的代币数.
 */
function balanceOf(address account) external view returns (uint256);

/**
 * @dev 转账 `amount` 单位代币，从调用者账户到另一账户 `to`.
 *
 * 如果成功，返回 `true`.
 *
 * 释放 {Transfer} 事件.
 */
function transfer(address to, uint256 amount) external returns (bool);

/**
 * @dev 返回`owner`账户授权给`spender`账户的额度，默认为0。
 *
 * 当{approve} 或 {transferFrom} 被调用时，`allowance`会改变.
 */
function allowance(address owner, address spender) external view returns (uint256);

/**
 * @dev 调用者账户给`spender`账户授权 `amount`数量代币。
 *
 * 如果成功，返回 `true`.
 *
 * 释放 {Approval} 事件.
 */
function approve(address spender, uint256 amount) external returns (bool);

/**
 * @dev 通过授权机制，从`from`账户向`to`账户转账`amount`数量代币。转账的部分会从调用者的`allowance`中扣除。
 *
 * 如果成功，返回 `true`.
 *
 * 释放 {Transfer} 事件.
 */
function transferFrom(
    address from,
    address to,
    uint256 amount
) external returns (bool);
```

## 参考文档

* https://ethereum.org/zh/developers/docs/standards/tokens/erc-20/