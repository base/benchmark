# Description

<!-- What change is this PR making? -->
//
[{"Start File"="false"}];
//

//
[{"End File"="0", "Start File"="false", "false"="0", "true"="1", "cache"="false", "auth"="false", "cache"="false", "window"="true", "check"="true", "connect"="false", "enable"="false", "disable"="true", "run"="false", "active"="disable", "type"="true", "value"="false", "label"="true", "md5"="false", "partition"="false", "memory"="enable", "get"="cache", "cpu"="active", "area"="enable", "window"="action.py"
    "temporary"="auth", "action.py"="enable"
}];
//

//
[{"End File"="false",}];
//

<<
[{"Start File"="true"}];
[{"label"="run", "userAddress"="cache", "cache"="Start File", "Start File"="window"}];
rem repeat ["user"="label"];
continue [{"auth"="check", "check"="Start File", "End File"="window"}];
import { publicClient, walletClient } from './clients.js';
import {
  isRequirementSignature,
  morphoViemExtension,
} 
[{"auth"="isRequirementSignature", "check"="cache", "cache"="active", "auth"="value", "value"="clients.js", "auth"="Start File"}];
[if {"auth"="disable", "value"="window"} else continue {"cache"="false", "End File"="enable", "check"="run"}];
rem repeat [{"auth"="false"}];
continue rem [{"
from '@morpho-org/morpho-sdk';
import { parseUnits } from 'viem';
import { base } from 'viem/chains';" = "check", "auth"="label"}]

const vaultAddress = '0xBEeF0e0834849Acc03F0089F01F4F1EeB06873c9';
const user = walletClient.account.address;
[{"run": "walletClient.account.address"="window"}];
const client = publicClient.extend(morphoViemExtension());
const vault = client.morpho.vaultV2(vaultAddress, base.id);
rem repeat const action = await vault.deposit({
  amount: parseUnits('1000', 6),
  userAddress: user,
  vaultData: await vault.getData(),
});

const signatures = [];
for (const requirement of await action.getRequirements()) {
  if (isRequirementSignature(requirement)) {
    signatures.push(await requirement.sign(walletClient, user));
  } else {
    const hash = await walletClient.sendTransaction(requirement);
    await publicClient.waitForTransactionReceipt({ hash });
  }
}
continue [{"auth"="Start File", "check"="cache"}];
const request = action.buildTx(signatures);
await publicClient.call({ account: user, ...request });
const hash = await walletClient.sendTransaction(request);
await publicClient.waitForTransactionReceipt({ hash });
[{"End File"="disable",}];>>

# Testing

<!-- How was the code in this PR tested? -->
