/*
	Copyright 2021 SANGFOR TECHNOLOGIES

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/
/*
   Copyright 2016 GitHub Inc.
	 See https://gitee.com/opengauss/ham4db/blob/master/LICENSE
*/

package cli

import (
	"fmt"
	"strings"
)

const AppPrompt string = `
ham4db [-c command] [-i instance] [-d destination] [--verbose|--debug] [... cli ] | http

try this:
    Run ham4db in HTTP mode:

        ham4db --debug http

    See all possible commands:

        ham4db help

    Detailed help for a given command (e.g. "relocate")

        ham4db help relocate
`

var CommandHelp map[string]string

func init() {
	CommandHelp = make(map[string]string)
	CommandHelp["relocate"] = `
  Relocate a replica beneath another (destination) instance. The choice of destination is almost arbitrary;
  it must not be a child/descendant of the instance, but otherwise it can be anywhere, and can be a normal replica
  or a binlog server. will choose the best course of action to relocate the replica.
  No action taken when destination instance cannot act as master (e.g. has no binary logs, is of incompatible version, incompatible binlog format etc.)
  Examples:

  ham4db -c relocate -i replica.to.relocate.com -d instance.that.becomes.its.master

  ham4db -c relocate -d destination.instance.that.becomes.its.master
      -i not given, implicitly assumed local hostname

  (this command was previously named "relocate-below")
  `
	CommandHelp["relocate-replicas"] = `
  Relocates all or part of the replicas of a given instance under another (destination) instance. This is
  typically much faster than relocating replicas one by one.
  Chooses the best course of action to relocation the replicas. It may choose a multi-step operations.
  Some replicas may succeed and some may fail the operation.
  The instance (replicas' master) itself may be crashed or inaccessible. It is not contacted throughout the operation.
  Examples:

  ham4db -c relocate-replicas -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master

  ham4db -c relocate-replicas -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master --pattern=regexp.filter
      only apply to those instances that match given regex
  `
	CommandHelp["move-up-replicas"] = `
  Moves replicas of the given instance one level up the topology, making them siblings of given instance.
  This is a (faster) shortcut to executing move-up on all replicas of given instance.
  Examples:

  ham4db -c move-up-replicas -i replica.whose.subreplicas.will.move.up.com[:3306]

  ham4db -c move-up-replicas -i replica.whose.subreplicas.will.move.up.com[:3306] --pattern=regexp.filter
      only apply to those instances that match given regex
	`
	CommandHelp["move-below"] = `
  Moves a replica beneath its sibling. Both replicas must be actively replicating from same master.
  The sibling will become instance's master. No action taken when sibling cannot act as master
  (e.g. has no binary logs, is of incompatible version, incompatible binlog format etc.)
  Example:

  ham4db -c move-below -i replica.to.move.com -d sibling.replica.under.which.to.move.com

  ham4db -c move-below -d sibling.replica.under.which.to.move.com
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["move-equivalent"] = `
  Moves a replica beneath another server, based on previously recorded "equivalence coordinates". Such coordinates
  are obtained whenever ham4db issues a CHANGE MASTER TO. The "before" and "after" masters coordinates are
  persisted. In such cases where the newly relocated replica is unable to replicate (e.g. firewall issues) it is then
  easy to revert the relocation via "move-equivalent".
  The command works if and only if ham4db has an exact mapping between the replica's current replication coordinates
  and some other coordinates.
  Example:

  ham4db -c move-equivalent -i replica.to.revert.master.position.com -d master.to.move.to.com
	`
	CommandHelp["take-siblings"] = `
  Turn all siblings of a replica into its sub-replicas. No action taken for siblings that cannot become
  replicas of given instance (e.g. incompatible versions, binlog format etc.). This is a (faster) shortcut
  to executing move-below for all siblings of the given instance. Example:

  ham4db -c take-siblings -i replica.whose.siblings.will.move.below.com
	`
	CommandHelp["take-master"] = `
  Turn an instance into a master of its own master; essentially switch the two. DownstreamKeyMap of each of the two
  involved instances are unaffected, and continue to replicate as they were.
  The instance's master must itself be a replica. It does not necessarily have to be actively replicating.

  ham4db -c take-master -i replica.that.will.switch.places.with.its.master.com
	`
	CommandHelp["repoint"] = `
  Make the given instance replicate from another instance without changing the binglog coordinates. There
  are little sanity checks to this and this is a risky operation. Use cases are: a rename of the master's
  host, a corruption in relay-logs, move from beneath MaxScale & Binlog-server. Examples:

  ham4db -c repoint -i replica.to.operate.on.com -d new.master.com

  ham4db -c repoint -i replica.to.operate.on.com
      The above will repoint the replica back to its existing master without change

  ham4db -c repoint
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["repoint-replicas"] = `
  Repoint all replicas of given instance to replicate back from the instance. This is a convenience method
  which implies a one-by-one "repoint" command on each replica.

  ham4db -c repoint-replicas -i instance.whose.replicas.will.be.repointed.com

  ham4db -c repoint-replicas
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["make-co-master"] = `
  Create a master-master replication. Given instance is a replica which replicates directly from a master.
  The master is then turned to be a replica of the instance. The master is expected to not be a replica.
  The read_only property of the slve is unaffected by this operation. Examples:

  ham4db -c make-co-master -i replica.to.turn.into.co.master.com

  ham4db -c make-co-master
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["repl-get-candidate"] = `
  Information command suggesting the most up-to-date replica of a given instance, which can be promoted
  as local master to its siblings. If replication is up and running, this command merely gives an
  estimate, since replicas advance and progress continuously in different pace. If all replicas of given
  instance have broken replication (e.g. because given instance is dead), then this command provides
  with a definitve candidate, which could act as a replace master. See also regroup-replicas. Example:

  ham4db -c repl-get-candidate -i instance.with.replicas.one.of.which.may.be.candidate.com
	`
	CommandHelp["regroup-replicas-bls"] = `
  Given an instance that has Binlog Servers for replicas, promote one such Binlog Server over its other
  Binlog Server siblings.

  Example:

  ham4db -c regroup-replicas-bls -i instance.with.binlog.server.replicas.com

  --debug is your friend.
	`
	CommandHelp["move-gtid"] = `
  Move a replica beneath another (destination) instance. Will reject the operation if GTID is
  not enabled on the replica, or is not supported by the would-be master.
  You may try and move the replica under any other instance; there are no constraints on the family ties the
  two may have, though you should be careful as not to try and replicate from a descendant (making an
  impossible loop).
  Examples:

  ham4db -c move-gtid -i replica.to.move.com -d instance.that.becomes.its.master

  ham4db -c match -d destination.instance.that.becomes.its.master
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["move-replicas-gtid"] = `
  Moves all replicas of a given instance under another (destination) instance using GTID. This is a (faster)
  shortcut to moving each replica via "move-gtid".
  Will only move those replica configured with GTID (either Oracle or MariaDB variants) and under the
  condition the would-be master supports GTID.
  Examples:

  ham4db -c move-replicas-gtid -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master

  ham4db -c move-replicas-gtid -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master --pattern=regexp.filter
      only apply to those instances that match given regex
	`
	CommandHelp["regroup-replicas-gtid"] = `
  Given an instance (possibly a crashed one; it is never being accessed), pick one of its replica and make it
  local master of its siblings, using GTID. The rules are similar to those in the "regroup-replicas" command.
  Example:

  ham4db -c regroup-replicas-gtid -i instance.with.gtid.and.replicas.one.of.which.will.turn.local.master.if.possible

  --debug is your friend.
	`
	CommandHelp["match"] = `
  Matches a replica beneath another (destination) instance. The choice of destination is almost arbitrary;
  it must not be a child/descendant of the instance. But otherwise they don't have to be direct siblings,
  and in fact (if you know what you're doing), they don't actually have to belong to the same topology.
  The operation expects the relocated instance to be "behind" the destination instance. It only finds out
  whether this is the case by the end; the operation is cancelled in the event this is not the case.
  No action taken when destination instance cannot act as master (e.g. has no binary logs, is of incompatible version, incompatible binlog format etc.)
  Examples:

  ham4db -c match -i replica.to.relocate.com -d instance.that.becomes.its.master

  ham4db -c match -d destination.instance.that.becomes.its.master
      -i not given, implicitly assumed local hostname

  (this command was previously named "match-below")
	`
	CommandHelp["match-replicas"] = `
  Matches all replicas of a given instance under another (destination) instance. This is a (faster) shortcut
  to matching said replicas one by one under the destination instance. In fact, this bulk operation is highly
  optimized and can execute in orders of magnitue faster, depeding on the nu,ber of replicas involved and their
  respective position behind the instance (the more replicas, the more savings).
  The instance itself may be crashed or inaccessible. It is not contacted throughout the operation. Examples:

  ham4db -c match-replicas -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master

  ham4db -c match-replicas -i instance.whose.replicas.will.relocate -d instance.that.becomes.their.master --pattern=regexp.filter
      only apply to those instances that match given regex

  (this command was previously named "multi-match-replicas")
	`
	CommandHelp["match-up"] = `
  Transport the replica one level up the hierarchy, making it child of its grandparent. This is
  similar in essence to move-up, only based on Pseudo-GTID. The master of the given instance
  does not need to be alive or connected (and could in fact be crashed). It is never contacted.
  Grandparent instance must be alive and accessible.
  Examples:

  ham4db -c match-up -i replica.to.match.up.com:3306

  ham4db -c match-up
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["match-up-replicas"] = `
  Matches replicas of the given instance one level up the topology, making them siblings of given instance.
  This is a (faster) shortcut to executing match-up on all replicas of given instance. The instance need
  not be alive / accessib;e / functional. It can be crashed.
  Example:

  ham4db -c match-up-replicas -i replica.whose.subreplicas.will.match.up.com

  ham4db -c match-up-replicas -i replica.whose.subreplicas.will.match.up.com[:3306] --pattern=regexp.filter
      only apply to those instances that match given regex
	`
	CommandHelp["rematch"] = `
  Reconnect a replica onto its master, via PSeudo-GTID. The use case for this operation is a non-crash-safe
  replication configuration (e.g. MySQL 5.5) with sync_binlog=1 and log_slave_updates. This operation
  implies crash-safe-replication and makes it possible for the replica to reconnect. Example:

  ham4db -c rematch -i replica.to.rematch.under.its.master
	`
	CommandHelp["regroup-replicas"] = `
  Given an instance (possibly a crashed one; it is never being accessed), pick one of its replica and make it
  local master of its siblings, using Pseudo-GTID. It is uncertain that there *is* a replica that will be able to
  become master to all its siblings. But if there is one, ham4db will pick such one. There are many
  constraints, most notably the replication positions of all replicas, whether they use log_slave_updates, and
  otherwise version compatabilities etc.
  As many replicas that can be regrouped under promoted slves are operated on. The rest are untouched.
  This command is useful in the event of a crash. For example, in the event that a master dies, this operation
  can promote a candidate replacement and set up the remaining topology to correctly replicate from that
  replacement replica. Example:

  ham4db -c regroup-replicas -i instance.with.replicas.one.of.which.will.turn.local.master.if.possible

  --debug is your friend.
	`

	CommandHelp["enable-gtid"] = `
  If possible, enable GTID replication. This works on Oracle (>= 5.6, gtid-mode=1) and MariaDB (>= 10.0).
  Replication is stopped for a short duration so as to reconfigure as GTID. In case of error replication remains
  stopped. Example:

  ham4db -c enable-gtid -i replica.compatible.with.gtid.com
	`
	CommandHelp["disable-gtid"] = `
  Assuming replica replicates via GTID, disable GTID replication and resume standard file:pos replication. Example:

  ham4db -c disable-gtid -i replica.replicating.via.gtid.com
	`
	CommandHelp["reset-master-gtid-remove-own-uuid"] = `
  Assuming GTID is enabled, Reset master on instance, remove GTID entries generated by the instance.
  This operation is only allowed on Oracle-GTID enabled servers that have no replicas.
  Is is used for cleaning up the GTID mess incurred by mistakenly issuing queries on the replica (even such
  queries as "FLUSH ENGINE LOGS" that happen to write to binary logs). Example:

  ham4db -c reset-master-gtid-remove-own-uuid -i replica.running.with.gtid.com
	`
	CommandHelp["stop-slave"] = `
  Issues a STOP SLAVE; command. Example:

  ham4db -c stop-slave -i replica.to.be.stopped.com
	`
	CommandHelp["start-slave"] = `
  Issues a START SLAVE; command. Example:

  ham4db -c start-slave -i replica.to.be.started.com
	`
	CommandHelp["restart-slave"] = `
  Issues STOP SLAVE + START SLAVE; Example:

  ham4db -c restart-slave -i replica.to.be.started.com
	`
	CommandHelp["skip-query"] = `
  On a failed replicating replica, skips a single query and attempts to resume replication.
  Only applies when the replication seems to be broken on SQL thread (e.g. on duplicate
  key error). Also works in GTID mode. Example:

  ham4db -c skip-query -i replica.with.broken.sql.thread.com
	`
	CommandHelp["reset-slave"] = `
  Issues a RESET SLAVE command. Destructive to replication. Example:

  ham4db -c reset-slave -i replica.to.reset.com
	`
	CommandHelp["detach-replica"] = `
  Stops replication and modifies binlog position into an impossible, yet reversible, value.
  This effectively means the replication becomes broken. See reattach-replica. Example:

  ham4db -c detach-replica -i replica.whose.replication.will.break.com

  Issuing this on an already detached replica will do nothing.
	`
	CommandHelp["reattach-replica"] = `
  Undo a detach-replica operation. Reverses the binlog change into the original values, and
  resumes replication. Example:

  ham4db -c reattach-replica -i detahced.replica.whose.replication.will.amend.com

  Issuing this on an attached (i.e. normal) replica will do nothing.
	`
	CommandHelp["detach-replica-master-host"] = `
  Stops replication and modifies Master_Host into an impossible, yet reversible, value.
  This effectively means the replication becomes broken. See reattach-replica-master-host. Example:

  ham4db -c detach-replica-master-host -i replica.whose.replication.will.break.com

  Issuing this on an already detached replica will do nothing.
	`
	CommandHelp["reattach-replica-master-host"] = `
  Undo a detach-replica-master-host operation. Reverses the hostname change into the original value, and
  resumes replication. Example:

  ham4db -c reattach-replica-master-host -i detahced.replica.whose.replication.will.amend.com

  Issuing this on an attached (i.e. normal) replica will do nothing.
	`
	CommandHelp["restart-slave-statements"] = `
	Prints a list of statements to execute to stop then restore replica to same execution state.
	Provide --statement for injected statement.
	This is useful for issuing a command that can only be executed while replica is stopped. Such
	commands are any of CHANGE MASTER TO.
	Will not execute given commands, only print them as courtesy. It may not have
	the privileges to execute them in the first place. Example:

	ham4db -c restart-slave-statements -i some.replica.com -statement="change master to master_heartbeat_period=5"
	`

	CommandHelp["set-read-only"] = `
  Turn an instance read-only, via SET GLOBAL read_only := 1. Examples:

  ham4db -c set-read-only -i instance.to.turn.read.only.com

  ham4db -c set-read-only
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["set-writeable"] = `
  Turn an instance writeable, via SET GLOBAL read_only := 0. Example:

  ham4db -c set-writeable -i instance.to.turn.writeable.com

  ham4db -c set-writeable
      -i not given, implicitly assumed local hostname
	`

	CommandHelp["flush-binary-logs"] = `
  Flush binary logs on an instance. Examples:

  ham4db -c flush-binary-logs -i instance.with.binary.logs.com

  ham4db -c flush-binary-logs -i instance.with.binary.logs.com --binlog=mysql-bin.002048
      Flushes binary logs until reaching given number. Fails when current number is larger than input
	`
	CommandHelp["purge-binary-logs"] = `
  Purge binary logs on an instance. Examples:

  ham4db -c purge-binary-logs -i instance.with.binary.logs.com --binlog mysql-bin.002048

      Purges binary logs until given log
	`
	CommandHelp["last-pseudo-gtid"] = `
  Information command; an authoritative way of detecting whether a Pseudo-GTID event exist for an instance,
  and if so, output the last Pseudo-GTID entry and its location. Example:

  ham4db -c last-pseudo-gtid -i instance.with.possible.pseudo-gtid.injection
	`
	CommandHelp["find-binlog-entry"] = `
  Get binlog file:pos of entry given by --pattern (exact full match, not a regular expression) in a given instance.
  This will search the instance's binary logs starting with most recent, and terminate as soon as an exact match is found.
  The given input is not a regular expression. It must fully match the entry (not a substring).
  This is most useful when looking for uniquely identifyable values, such as Pseudo-GTID. Example:

  ham4db -c find-binlog-entry -i instance.to.search.on.com --pattern "insert into my_data (my_column) values ('distinct_value_01234_56789')"

      Prints out the binlog file:pos where the entry is found, or errors if unfound.
	`
	CommandHelp["correlate-binlog-pos"] = `
  Given an instance (-i) and binlog coordinates (--binlog=file:pos), find the correlated coordinates in another instance (-d).
  "Correlated coordinates" are those that present the same point-in-time of sequence of binary log events, untangling
  the mess of different binlog file:pos coordinates on different servers.
  This operation relies on Pseudo-GTID: your servers must have been pre-injected with PSeudo-GTID entries as these are
  being used as binlog markers in the correlation process.
  You must provide a valid file:pos in the binlogs of the source instance (-i), and in response get the correlated
  coordinates in the binlogs of the destination instance (-d). This operation does not work on relay logs.
  Example:

  ham4db -c correlate-binlog-pos  -i instance.with.binary.log.com --binlog=mysql-bin.002366:14127 -d other.instance.with.binary.logs.com

      Prints out correlated coordinates, e.g.: "mysql-bin.002302:14220", or errors out.
	`

	CommandHelp["submit-pool-instances"] = `
  Submit a pool name with a list of instances in that pool. This removes any previous instances associated with
  that pool. Expecting comma delimited list of instances

  ham4db -c submit-pool-instances --pool name_of_pool -i pooled.instance1.com,pooled.instance2.com:3306,pooled.instance3.com
	`
	CommandHelp["cluster-pool-instances"] = `
  List all pools and their associated instances. Output is in tab delimited format, and lists:
  cluster_name, cluster_alias, pool_name, pooled instance
  Example:

  ham4db -c cluster-pool-instances
	`
	CommandHelp["which-heuristic-cluster-pool-instances"] = `
	List instances belonging to a cluster, which are also in some pool or in a specific given pool.
	Not all instances are listed: unreachable, downtimed instances ar left out. Only those that should be
	responsive and healthy are listed. This serves applications in getting information about instances
	that could be queried (this complements a proxy behavior in providing the *list* of instances).
	Examples:

	ham4db -c which-heuristic-cluster-pool-instances --alias mycluster
			Get the instances of a specific cluster, no specific pool

	ham4db -c which-heuristic-cluster-pool-instances --alias mycluster --pool some_pool
			Get the instances of a specific cluster and which belong to a given pool

	ham4db -c which-heuristic-cluster-pool-instances -i instance.belonging.to.a.cluster
			Cluster inferred by given instance

	ham4db -c which-heuristic-cluster-pool-instances
			Cluster inferred by local hostname
	`

	CommandHelp["find"] = `
  Find instances whose hostname matches given regex pattern. Example:

  ham4db -c find -pattern "backup.*us-east"
	`
	CommandHelp["clusters"] = `
  List all clusters known to ham4db. A cluster (aka topology, aka chain) is identified by its
  master (or one of its master if more than one exists). Example:

  ham4db -c clusters
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["all-clusters-masters"] = `
  List of writeable masters, one per cluster.
	For most single-master topologies, this is trivially the master.
	For active-active master-master topologies, this ensures only one of
	the masters is returned. Example:

        ham4db -c all-clusters-masters
	`
	CommandHelp["topology"] = `
  Show an ascii-graph of a replication topology, given a member of that topology. Example:

  ham4db -c topology -i instance.belonging.to.a.topology.com

  ham4db -c topology
      -i not given, implicitly assumed local hostname

  Instance must be already known to ham4db. Topology is generated by ham4db's mapping
  and not from synchronuous investigation of the instances. The generated topology may include
  instances that are dead, or whose replication is broken.
	`
	CommandHelp["all-instances"] = `
  List the complete known set of instances. Similar to '-c find -pattern "."' Example:

    ham4db -c all-instances
	`
	CommandHelp["which-instance"] = `
  Output the fully-qualified hostname:port representation of the given instance, or error if unknown
  to ham4db. Examples:

  ham4db -c which-instance -i instance.to.check.com

  ham4db -c which-instance
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["which-cluster"] = `
  Output the name of the cluster an instance belongs to, or error if unknown to ham4db. Examples:

  ham4db -c which-cluster -i instance.to.check.com

  ham4db -c which-cluster
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["list-cluster-instance"] = `
  Output the list of instances participating in same cluster as given instance; output is one line
  per instance, in hostname:port format. Examples:

  ham4db -c list-cluster-instance -i instance.to.check.com

  ham4db -c list-cluster-instance
      -i not given, implicitly assumed local hostname

  ham4db -c list-cluster-instance -alias some_alias
      assuming some_alias is a known cluster alias (see ClusterNameToAlias or DetectClusterAliasQuery configuration)
	`
	CommandHelp["which-cluster-domain"] = `
  Output the domain name of given cluster, indicated by instance or alias. This depends on
	the DetectClusterDomainQuery configuration. Example:

  ham4db -c which-cluster-domain -i instance.to.check.com

  ham4db -c which-cluster-domain
      -i not given, implicitly assumed local hostname

  ham4db -c which-cluster-domain -alias some_alias
      assuming some_alias is a known cluster alias (see ClusterNameToAlias or DetectClusterAliasQuery configuration)
	`
	CommandHelp["which-heuristic-domain-instance"] = `
	Returns the instance associated as the cluster's writer with a cluster's domain name.
	Given a cluster, ham4db looks for the domain name indicated by this cluster, and proceeds to search for
	a stord key-value attribute for that domain name. This would be the writer host for the given domain.
	See also set-heuristic-domain-instance, this is meant to be a temporary service mimicking in micro-scale a
	service discovery functionality.
	Example:

	ham4db -c which-heuristic-domain-instance -alias some_alias
		Detects the domain name for given cluster, reads from key-value store the writer host associated with the domain name.

	ham4db -c which-heuristic-domain-instance -i instance.of.some.cluster
		Cluster is inferred by a member instance (the instance is not necessarily the master)
	`
	CommandHelp["cluster-master"] = `
	Output the name of the active master in a given cluster, indicated by instance or alias.
	An "active" master is one that is writable and is not marked as downtimed due to a topology recovery.
	Examples:

  ham4db -c cluster-master -i instance.to.check.com

  ham4db -c cluster-master
      -i not given, implicitly assumed local hostname

  ham4db -c cluster-master -alias some_alias
      assuming some_alias is a known cluster alias (see ClusterNameToAlias or DetectClusterAliasQuery configuration)
	`
	CommandHelp["which-cluster-osc-replicas"] = `
  Output a list of replicas in same cluster as given instance, that would server as good candidates as control replicas
  for a pt-online-schema-change operation.
  Those replicas would be used for replication delay so as to throtthe osc operation. Selected replicas will include,
  where possible: intermediate masters, their replicas, 3rd level replicas, direct non-intermediate-master replicas.

  ham4db -c which-cluster-osc-replicas -i instance.to.check.com

  ham4db -c which-cluster-osc-replicas
      -i not given, implicitly assumed local hostname

  ham4db -c which-cluster-osc-replicas -alias some_alias
      assuming some_alias is a known cluster alias (see ClusterNameToAlias or DetectClusterAliasQuery configuration)
	`
	CommandHelp["which-lost-in-recovery"] = `
	List instances marked as downtimed for being lost in a recovery process. The output of this command lists
  "lost" instances that probably should be recycled.
	The topology recovery process injects a magic hint when downtiming lost instances, that is picked up
	by this command. Examples:

	ham4db -c which-lost-in-recovery
			Lists all heuristically-recent known lost instances
	`
	CommandHelp["instance-master-identify"] = `
  Output the fully-qualified hostname:port representation of a given instance's master. Examples:

  ham4db -c instance-master-identify -i a.known.replica.com

  ham4db -c instance-master-identify
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["repl-list-replica"] = `
  Output the fully-qualified hostname:port list of replicas (one per line) of a given instance (or empty
  list if instance is not a master to anyone). Examples:

  ham4db -c repl-list-replica -i a.known.instance.com

  ham4db -c repl-list-replica
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["list-cluster-heuristic-lag"] = `
  For a given cluster (indicated by an instance or alias), output a heuristic "representative" lag of that cluster.
  The output is obtained by examining the replicas that are member of "which-cluster-osc-replicas"-command, and
  getting the maximum replica lag of those replicas. Recall that those replicas are a subset of the entire cluster,
  and that they are ebing polled periodically. Hence the output of this command is not necessarily up-to-date
  and does not represent all replicas in cluster. Examples:

  ham4db -c list-cluster-heuristic-lag -i instance.that.is.part.of.cluster.com

  ham4db -c list-cluster-heuristic-lag
      -i not given, implicitly assumed local host, cluster implied

  ham4db -c list-cluster-heuristic-lag -alias some_alias
      assuming some_alias is a known cluster alias (see ClusterNameToAlias or DetectClusterAliasQuery configuration)
	`
	CommandHelp["instance-status"] = `
  Output short status on a given instance (name, replication status, noteable configuration). Example2:

  ham4db -c instance-status -i instance.to.investigate.com

  ham4db -c instance-status
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["snapshot-topologies"] = `
  Take a snapshot of existing topologies. This will record minimal replication topology data: the identity
  of an instance, its master and its cluster.
  Taking a snapshot later allows for reviewing changes in topologies. One might wish to invoke this command
  on a daily basis, and later be able to solve questions like 'where was this instacne replicating from before
  we moved it?', 'which instances were replication from this instance a week ago?' etc. Example:

  ham4db -c snapshot-topologies
	`

	CommandHelp["discover"] = `
  Request that ham4db cotacts given instance, reads its status, and upsert it into
  ham4db's respository. Examples:

  ham4db -c discover -i instance.to.discover.com:3306

  ham4db -c discover -i cname.of.instance

  ham4db -c discover
      -i not given, implicitly assumed local hostname

  Will resolve CNAMEs and VIPs.
	`
	CommandHelp["forget"] = `
  Request that ham4db removed given instance from its repository. If the instance is alive
  and connected through replication to otherwise known and live instances, ham4db will
  re-discover it by nature of its discovery process. Instances are auto-removed via config's
  UnseenAgentForgetHours. If you happen to know a machine is decommisioned, for example, it
  can be nice to remove it from the repository before it auto-expires. Example:

  ham4db -c forget -i instance.to.forget.com

  Will *not* resolve CNAMEs and VIPs for given instance.
	`
	CommandHelp["begin-maintenance"] = `
  Request a maintenance lock on an instance. Topology changes require placing locks on the minimal set of
  affected instances, so as to avoid an incident of two uncoordinated operations on a smae instance (leading
  to possible chaos). Locks are placed in the backend database, and so multiple ham4db instances are safe.
  Operations automatically acquire locks and release them. This command manually acquires a lock, and will
  block other operations on the instance until lock is released.
  Note that ham4db automatically assumes locks to be expired after MaintenanceExpireMinutes (hard coded value).
  Examples:

  ham4db -c begin-maintenance -i instance.to.lock.com --duration=3h --reason="load testing; do not disturb"
      accepted duration format: 10s, 30m, 24h, 3d, 4w

  ham4db -c begin-maintenance -i instance.to.lock.com --reason="load testing; do not disturb"
      --duration not given; default to MaintenanceExpireMinutes (hard coded value)
	`
	CommandHelp["end-maintenance"] = `
  Remove maintenance lock; such lock may have been gained by an explicit begin-maintenance command implicitly
  by a topology change. You should generally only remove locks you have placed manually; ham4db will
  automatically expire locks after MaintenanceExpireMinutes (hard coded value).
  Example:

  ham4db -c end-maintenance -i locked.instance.com
	`
	CommandHelp["begin-downtime"] = `
  Mark an instance as downtimed. A downtimed instance is assumed to be taken care of, and recovery-analysis does
  not apply for such an instance. As result, no recommendation for recovery, and no automated-recovery are issued
  on a downtimed instance.
  Downtime is different than maintanence in that it places no lock (mainenance uses an exclusive lock on the instance).
  It is OK to downtime an instance that is already downtimed -- the new begin-downtime command will override whatever
  previous downtime attributes there were on downtimes instance.
  Note that ham4db automatically assumes downtime to be expired after MaintenanceExpireMinutes (hard coded value).
  Examples:

  ham4db -c begin-downtime -i instance.to.downtime.com --duration=3h --reason="dba handling; do not do recovery"
      accepted duration format: 10s, 30m, 24h, 3d, 4w

  ham4db -c begin-downtime -i instance.to.lock.com --reason="dba handling; do not do recovery"
      --duration not given; default to MaintenanceExpireMinutes (hard coded value)
	`
	CommandHelp["end-downtime"] = `
  Indicate an instance is no longer downtimed. Typically you should not need to use this since
  a downtime is always bounded by a duration and auto-expires. But you may use this to forcibly
  indicate the active downtime should be expired now.
  Example:

  ham4db -c end-downtime -i downtimed.instance.com
	`

	CommandHelp["recover"] = `
  Do auto-recovery given a dead instance. Chooses the best course of action.
  The given instance must be acknowledged as dead and have replicas, or else there's nothing to do.
  See "replication-analysis" command.
  Executes external processes as configured by *Processes variables.
  --debug is your friend. Example:

  ham4db -c recover -i dead.instance.com --debug
	`
	CommandHelp["recover-lite"] = `
  Do auto-recovery given a dead instance. Chooses the best course of action, exactly
  as in "-c recover". Orchestratir will *not* execute external processes.

  ham4db -c recover-lite -i dead.instance.com --debug
	`
	CommandHelp["force-master-failover"] = `
  Forcibly begin a master failover process, even if ham4db does not see anything wrong
  in particular with the master.
  - This will not work in a master-master configuration
	- Just treats this command as a DeadMaster failover scenario
  - Will issue all relevant pre-failover and post-failover external processes.
  - Will not attempt to recover/reconnect the old master
	`
	CommandHelp["force-master-takeover"] = `
	Forcibly discard master and promote another (direct child) instance instead, even if everything is running well.
	This allows for planned switchover.
	NOTE:
	- You must specify the instance to promote via "-d"
	- Promoted instance must be a direct child of the existing master
	- This will not work in a master-master configuration
	- Just treats this command as a DeadMaster failover scenario
	- It is STRONGLY suggested that you first relocate everything below your chosen instance-to-promote.
	  It *is* a planned failover thing.
	- Otherwise ham4db will do its thing in moving instances around, hopefully promoting your requested
	  server on top.
	- Will issue all relevant pre-failover and post-failover external processes.
	- In this command ham4db will not issue 'SET GLOBAL read_only=1' on the existing master, nor will
	  it issue a 'FLUSH TABLES WITH READ LOCK'. Please see the 'graceful-master-takeover' command.
	Examples:

	ham4db -c force-master-takeover -alias mycluster -d immediate.child.of.master.com
			Indicate cluster by alias. Automatically figures out the master

	ham4db -c force-master-takeover -i instance.in.relevant.cluster.com -d immediate.child.of.master.com
			Indicate cluster by an instance. You don't structly need to specify the master, ham4db
			will infer the master's identify.
	`
	CommandHelp["graceful-master-takeover"] = `
	Gracefully discard master and promote another (direct child) instance instead, even if everything is running well.
	This allows for planned switchover.
	NOTE:
	- Promoted instance must be a direct child of the existing master
	- Promoted instance must be the *only* direct child of the existing master. It *is* a planned failover thing.
	- Will first issue a "set global read_only=1" on existing master
	- It will promote candidate master to the binlog positions of the existing master after issuing the above
	- There _could_ still be statements issued and executed on the existing master by SUPER users, but those are ignored.
	- Then proceeds to handle a DeadMaster failover scenario
	- Will issue all relevant pre-failover and post-failover external processes.
	Examples:

	ham4db -c graceful-master-takeover -alias mycluster
		Indicate cluster by alias. Automatically figures out the master and verifies it has a single direct replica

	ham4db -c force-master-takeover -i instance.in.relevant.cluster.com
		Indicate cluster by an instance. You don't structly need to specify the master, ham4db
		will infer the master's identify.
	`
	CommandHelp["replication-analysis"] = `
  Request an analysis of potential crash incidents in all known topologies.
  Output format is not yet stabilized and may change in the future. Do not trust the output
  for automated parsing. Use web API instead, at this time. Example:

  ham4db -c recovery-replication-analysis
	`
	CommandHelp["ack-cluster-recoveries"] = `
  Acknowledge recoveries for a given cluster; this unblocks pending future recoveries.
  Acknowledging a recovery requires a comment (supply via --reason). Acknowledgement clears the in-active-period
  flag for affected recoveries, which in turn affects any blocking recoveries.
  Multiple recoveries may be affected. Only unacknowledged recoveries will be affected.
  Examples:

  ham4db -c ack-cluster-recoveries -i instance.in.a.cluster.com --reason="dba has taken taken necessary steps"
       Cluster is indicated by any of its members. The recovery need not necessarily be on/to given instance.

  ham4db -c ack-cluster-recoveries -alias some_alias --reason="dba has taken taken necessary steps"
       Cluster indicated by alias
	`
	CommandHelp["ack-instance-recoveries"] = `
  Acknowledge recoveries for a given instance; this unblocks pending future recoveries.
  Acknowledging a recovery requires a comment (supply via --reason). Acknowledgement clears the in-active-period
  flag for affected recoveries, which in turn affects any blocking recoveries.
  Multiple recoveries may be affected. Only unacknowledged recoveries will be affected.
  Example:

  ham4db -c ack-cluster-recoveries -i instance.that.failed.com --reason="dba has taken taken necessary steps"
	`

	CommandHelp["register-candidate"] = `
  Indicate that a specific instance is a preferred candidate for master promotion. Upon a dead master
  recovery, ham4db will do its best to promote instances that are marked as candidates. However
  ham4db cannot guarantee this will always work. Issues like version compatabilities, binlog format
  etc. are limiting factors.
  You will want to mark an instance as a candidate when: it is replicating directly from the master, has
  binary logs and log_slave_updates is enabled, uses same binlog_format as its siblings, compatible version
  as its siblings. If you're using DataCenterPattern & EnvironmentPattern (see configuration),
  you would further wish to make sure you have a candidate in each data center.
  First promotes the best-possible replica, and only then replaces it with your candidate,
  and only if both in same datcenter and physical enviroment.
  An instance needs to continuously be marked as candidate, so as to make sure ham4db is not wasting
  time with stale instances. Periodically clears candidate-registration for instances that have
  not been registeres for over CandidateInstanceExpireMinutes (see config).
  Example:

  ham4db -c register-candidate -i candidate.instance.com

  ham4db -c register-candidate
      -i not given, implicitly assumed local hostname
	`
	CommandHelp["register-hostname-unresolve"] = `
  Assigns the given instance a virtual (aka "unresolved") name. When moving replicas under an instance with assigned
  "unresolve" name, ham4db issues a CHANGE MASTER TO MASTER_HOST='<the unresovled name instead of the fqdn>' ...
  This is useful in cases where your master is behind virtual IP (e.g. active/passive masters with shared storage or DRBD,
  e.g. binlog servers sharing common VIP).
  A "repoint" command is useful after "register-hostname-unresolve": you can repoint replicas of the instance to their exact
  same location, and ham4db will swap the fqdn of their master with the unresolved name.
  Such registration must be periodic. Automatically expires such registration after ExpiryHostnameResolvesMinutes.
  Example:

  ham4db -c register-hostname-unresolve -i instance.fqdn.com --hostname=virtual.name.com
	`
	CommandHelp["deregister-hostname-unresolve"] = `
  Explicitly deregister/dosassociate a hostname with an "unresolved" name. merely remvoes the association, but does
  not touch any replica at this point. A "repoint" command can be useful right after calling this command to change replica's master host
  name (assumed to be an "unresolved" name, such as a VIP) with the real fqdn of the master host.
  Example:

  ham4db -c deregister-hostname-unresolve -i instance.fqdn.com
	`
	CommandHelp["set-heuristic-domain-instance"] = `
	This is a temporary (sync your watches, watch for next ice age) command which registers the cluster domain name of a given cluster
	with the master/writer host for that cluster. It is a one-time-master-discovery operation.
	At this time ham4db may also act as a small & simple key-value store (recall the "temporary" indication).
	Master failover operations will overwrite the domain instance identity. so turns into a mini master-discovery
	service (I said "TEMPORARY"). Really there are other tools for the job. See also: which-heuristic-domain-instance
	Example:

	ham4db -c set-heuristic-domain-instance --alias some_alias
			Detects the domain name for given cluster, identifies the writer master of the cluster, associates the two in key-value store

	ham4db -c set-heuristic-domain-instance -i instance.of.some.cluster
			Cluster is inferred by a member instance (the instance is not necessarily the master)
	`

	CommandHelp["continuous"] = `
  Enter continuous mode, and actively poll for instances, diagnose problems, do maintenance etc.
  This type of work is typically done in HTTP mode. However nothing prevents ham4db from
  doing it in command line. Invoking with "continuous" will run indefinitely. Example:

  ham4db -c continuous
	`
	CommandHelp["active-nodes"] = `
	List ham4db nodes or processes that are actively running or have most recently
	executed. Output is in hostname:token format, where "token" is an internal unique identifier
	of an ham4db process. Example:

	ham4db -c active-nodes
	`
	CommandHelp["access-token"] = `
	When running HTTP with "AuthenticationMethod" : "token", receive a new access token.
	This token must be utilized within "AccessTokenUseExpirySeconds" and can then be used
	until "AccessTokenExpiryMinutes" have passed.
	In "token" authentication method a user is read-only unless able to provide with a fresh token.
	A token may only be used once (two users must get two distinct tokens).
	Submitting a token is done via "/web/access-token?publicToken=<received_token>". The token is then stored
	in HTTP cookie.

	ham4db -c access-token
	`
	CommandHelp["reset-hostname-resolve-cache"] = `
  Clear the hostname resolve cache; it will be refilled by following host discoveries

  ham4db -c reset-hostname-resolve-cache
	`
	CommandHelp["resolve"] = `
  Utility command to resolve a CNAME and return resolved hostname name. Example:

  ham4db -c resolve -i cname.to.resolve
	`
	CommandHelp["meta-redeploy-internal-db"] = `
	Force internal schema migration to current backend structure. keeps track of the deployed
	versions and will not reissue a migration for a version already deployed. Normally you should not use
	this command, and it is provided mostly for building and testing purposes. Nonetheless it is safe to
	use and at most it wastes some cycles.
	`

	for key := range CommandHelp {
		CommandHelp[key] = strings.Trim(CommandHelp[key], "\n")
	}
}

func HelpCommand(command string) {
	fmt.Println(
		fmt.Sprintf("%s:\n%s", command, CommandHelp[command]))
}
