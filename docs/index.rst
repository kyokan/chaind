Introduction
============

Building reliable blockchain infrastructure is a notoriously hard problem. As a result, most blockchain companies
don't run their own infrastructure, instead choosing to outsource it to a third-party. ``chaind`` makes deploying your
own infrastructure simpler and cheaper by acting as a proxy between the Internet and your blockchain nodes. Out of the
box, ``chaind`` provides the following features:

- Automatic failover to any blockchain node with an open RPC endpoint
- Intelligent request caching that takes chain reorgs into account
- RPC-aware request logging

Getting started
---------------

The simplest way to get started with ``chaind`` is to install one of our pre-built packages using your OS's package
manager. For more information, head on over to Installation and Configuration.

History of ``chaind``
---------------------

``chaind`` began in late 2018 as as an internal project at Kyokan. Since then, ``chaind`` has served millions of RPC
request on low-cost hardware.


.. toctree::
    :maxdepth: 2
    :caption: Contents:

    installation.rst
    configuration.rst