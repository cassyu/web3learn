const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("MyToken", function () {
  it("owner can mint to alice with uri", async function () {
    const [owner, alice] = await ethers.getSigners();
    const MyToken = await ethers.getContractFactory("MyToken");
    const nft = await MyToken.deploy(owner.address);
    await nft.waitForDeployment();

    const uri = "ipfs://bafyExample/0";
    await nft.connect(owner).safeMint(alice.address, uri);

    expect(await nft.ownerOf(0n)).to.equal(alice.address);
    expect(await nft.tokenURI(0n)).to.equal(uri);
  });

  it("non-owner cannot mint", async function () {
    const [owner, alice] = await ethers.getSigners();
    const MyToken = await ethers.getContractFactory("MyToken");
    const nft = await MyToken.deploy(owner.address);
    await nft.waitForDeployment();

    await expect(nft.connect(alice).safeMint(alice.address, "ipfs://x")).to.be.revertedWithCustomError(
      nft,
      "OwnableUnauthorizedAccount"
    );
  });
});
