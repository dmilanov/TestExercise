// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// ReleaseOracle is an Ethereum contract to store the current and previous
// versions of the go-ethereum implementation. Its goal is to allow Geth to
// check for new releases automatically without the need to consult a central
// repository.
//
// The contract takes a vote based approach on both assigning authorized signers
// as well as signing off on new Geth releases.
contract ReleaseOracle {
  // Votes is an internal data structure to count votes on a specific proposal
  struct Votes {
    address[] pass; // List of signers voting to pass a proposal
    address[] fail; // List of signers voting to fail a proposal
  }

  // Version is the version details of a particular Geth release
  struct Version {
    uint32  major;  // Major version component of the release
    uint32  minor;  // Minor version component of the release
    uint32  patch;  // Patch version component of the release
    bytes20 commit; // Git SHA1 commit hash of the release

    uint64  time;  // Timestamp of the release approval
    Votes   votes; // Votes that passed this release
  }

  // Oracle authorization details
  mapping(address => bool) authorized; // Set of accounts allowed to vote on updating the contract
  address[]                signers;    // List of addresses currently accepted as signers

  // Various proposals being voted on
  mapping(address => Votes) authProps; // Currently running user authorization proposals
  address[]                 authPend;  // List of addresses being voted on (map indexes)

  Version   verProp;  // Currently proposed release being voted on
  Version[] releases; // All the positively voted releases

  // isSigner is a modifier to authorize contract transactions.
  modifier isSigner() {
    if (authorized[msg.sender]) {
      _
    }
  }

  // Constructor to assign the creator as the sole valid signer.
  function ReleaseOracle() {
    authorized[msg.sender] = true;
    signers.push(msg.sender);
  }

  // Signers is an accessor method to retrieve all te signers (public accessor
  // generates an indexed one, not a retreive-all version).
  function Signers() constant returns(address[]) {
    return signers;
  }

  // AuthProposals retrieves the list of addresses that authorization proposals
  // are currently being voted on.
  function AuthProposals() constant returns(address[]) {
    return authPend;
  }

  // AuthVotes retrieves the current authorization votes for a particular user
  // to promote him into the list of signers, or demote him from there.
  function AuthVotes(address user) constant returns(address[] promote, address[] demote) {
    return (authProps[user].pass, authProps[user].fail);
  }

  // CurrentVersion retrieves the semantic version, commit hash and release time
  // of the currently votec active release.
  function CurrentVersion() constant returns (uint32 major, uint32 minor, uint32 patch, bytes20 commit, uint time) {
    var release = releases[releases.length - 1];

    return (release.major, release.minor, release.patch, release.commit, release.time);
  }

  // ProposedVersion retrieves the semantic version, commit hash and the current
  // votes for the next proposed release.
  function ProposedVersion() constant returns (uint32 major, uint32 minor, uint32 patch, bytes20 commit, address[] pass, address[] fail) {
    return (verProp.major, verProp.minor, verProp.patch, verProp.commit, verProp.votes.pass, verProp.votes.fail);
  }

  // Promote pitches in on a voting campaign to promote a new user to a signer
  // position.
  function Promote(address user) {
    updateSigner(user, true);
  }

  // Demote pitches in on a voting campaign to demote an authorized user from
  // its signer position.
  function Demote(address user) {
    updateSigner(user, false);
  }

  // Release votes for a particular version to be included as the next release.
  function Release(uint32 major, uint32 minor, uint32 patch, bytes20 commit) {
    updateRelease(major, minor, patch, commit, true);
  }

  // Nuke votes for the currently proposed version to not be included as the next
  // release. Nuking doesn't require a specific version number for simplicity.
  function Nuke() {
    updateRelease(0, 0, 0, 0, false);
  }

  // updateSigner marks a vote for changing the status of an Ethereum user, either
  // for or against the user being an authorized signer.
  function updateSigner(address user, bool authorize) isSigner {
    // Gather the current votes and ensure we don't double vote
    Votes votes = authProps[user];
    for (uint i = 0; i < votes.pass.length; i++) {
      if (votes.pass[i] == msg.sender) {
        return;
      }
    }
    for (i = 0; i < votes.fail.length; i++) {
      if (votes.fail[i] == msg.sender) {
        return;
      }
    }
    // If no authorization proposal is open, add the user to the index for later lookups
    if (votes.pass.length == 0 && votes.fail.length == 0) {
      authPend.push(user);
    }
    // Cast the vote and return if the proposal cannot be resolved yet
    if (authorize) {
      votes.pass.push(msg.sender);
      if (votes.pass.length <= signers.length / 2) {
        return;
      }
    } else {
      votes.fail.push(msg.sender);
      if (votes.fail.length <= signers.length / 2) {
        return;
      }
    }
    // Proposal resolved in our favor, execute whatever we voted on
    if (authorize && !authorized[user]) {
      authorized[user] = true;
      signers.push(user);
    } else if (!authorize && authorized[user]) {
      authorized[user] = false;

      for (i = 0; i < signers.length; i++) {
        if (signers[i] == user) {
          signers[i] = signers[signers.length - 1];
          signers.length--;
          break;
        }
      }
    }
    // Finally delete the resolved proposal, index and garbage collect
    delete authProps[user];

    for (i = 0; i < authPend.length; i++) {
      if (authPend[i] == user) {
        authPend[i] = authPend[authPend.length - 1];
        authPend.length--;
        break;
      }
    }
  }

  // updateRelease votes for a particular version to be included as the next release,
  // or for the currently proposed release to be nuked out.
  function updateRelease(uint32 major, uint32 minor, uint32 patch, bytes20 commit, bool release) isSigner {
    // Skip nuke votes if no proposal is pending
    if (!release && verProp.votes.pass.length == 0) {
      return;
    }
    // Mark a new release if no proposal is pending
    if (verProp.votes.pass.length == 0) {
      verProp.major  = major;
      verProp.minor  = minor;
      verProp.patch  = patch;
      verProp.commit = commit;
    }
    // Make sure positive votes match the current proposal
    if (release && (verProp.major != major || verProp.minor != minor || verProp.patch != patch || verProp.commit != commit)) {
      return;
    }
    // Gather the current votes and ensure we don't double vote
    Votes votes = verProp.votes;
    for (uint i = 0; i < votes.pass.length; i++) {
      if (votes.pass[i] == msg.sender) {
        return;
      }
    }
    for (i = 0; i < votes.fail.length; i++) {
      if (votes.fail[i] == msg.sender) {
        return;
      }
    }
    // Cast the vote and return if the proposal cannot be resolved yet
    if (release) {
      votes.pass.push(msg.sender);
      if (votes.pass.length <= signers.length / 2) {
        return;
      }
    } else {
      votes.fail.push(msg.sender);
      if (votes.fail.length <= signers.length / 2) {
        return;
      }
    }
    // Proposal resolved in our favor, execute whatever we voted on
    if (release) {
      verProp.time = uint64(now);
      releases.push(verProp);
      delete verProp;
    } else {
      delete verProp;
    }
  }
}
