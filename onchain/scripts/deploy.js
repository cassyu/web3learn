const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  const MyToken = await hre.ethers.getContractFactory("MyToken");
  const nft = await MyToken.deploy(deployer.address);
  await nft.waitForDeployment();

  const addr = await nft.getAddress();
  console.log("MyToken deployed to:", addr);
  console.log("Owner (initialOwner):", deployer.address);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
