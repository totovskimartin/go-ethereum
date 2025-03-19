const hre = require("hardhat");

async function main() {
  const currentTimestampInSeconds = Math.round(Date.now() / 1000);
  const unlockTime = currentTimestampInSeconds + 60;

  const lockedAmount = hre.ethers.parseEther("0.001");

  const lock = await hre.ethers.deployContract("Lock", [unlockTime], {
    value: lockedAmount,
  });

  await lock.waitForDeployment();

  console.log(
    `Lock with ${ethers.formatEther(
      lockedAmount
    )}ETH and unlock timestamp ${unlockTime} deployed to ${lock.target}`
  );

  let tx = lock.deploymentTransaction()

  const data = {
    "Transaction Hash": tx.hash,
    "From": tx.from,
    "To": tx.to,
    "Gas Limit": Number(tx.gasLimit),
    "Gas Price": Number(tx.gasPrice),
    "Block Number": tx.blockNumber
  } // Get the transaction hash

  console.log("Transaction Data:\n", data);


}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});