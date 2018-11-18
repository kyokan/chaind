Configuration
=============

Configuration file
------------------

``chaind`` is configured via a single ``chaind.toml`` file. When you install ``chaind`` using a package manager, a
default ``chaind.toml`` is placed in ``/etc/chaind``. It looks like this:

.. code-block:: toml

    eth_path = "eth"
    rpc_port = 8080
    log_level = "info"

    [log_auditor]
    log_file="/var/log/chaind_audit.log"

    [redis]
    url="localhost:6379"

    [[backend]]
    type="ETH"
    url="https://mainnet.infura.io/"
    name="infura"

    [[backend]]
    type="ETH"
    url="http://localhost:8545/"
    name="local"
    main=true

The only parts of the config file you'll need to change are the ``[[backend]]`` stanzas, since the default URLs are
only examples and won't work out of the box. These stanzas define which blockchain nodes ``chaind`` will be proxying to.
There can be an unlimited number of backends. Below, see a description of each available backend configuration
directive:

Backend configuration
---------------------

+------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| Key  | Description                                                                                                                                                                                                       |
+======+===================================================================================================================================================================================================================+
| type | The type of blockchain node. Currently, can only be ``ETH``, however in the future ``BTC`` (and potentially others) will be supported.                                                                            |
+------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| url  | The URL to the blockchain node. Can be ``http`` or ``https``.                                                                                                                                                     |
+------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| name | A name for the backend. Will appear in logs.                                                                                                                                                                      |
+------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| main | Optional. Defines whether or not ``chaind`` should proxy to this node by default. There can only be one ``main`` backend per ``type``. If ``main`` isn't specified, the first backend will be chosen as the main. |
+------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+

Server configuration
--------------------

The following directives are used to configure ``chaind`` itself:

+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+
| Key                          | Description                                                                                                                                    |
+==============================+================================================================================================================================================+
| eth_path                     | The HTTP path at which to serve Ethereum RPC requests. Defaults to ``eth``. Note that this value does not include a leading or trailing slash. |
+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+
| rpc_port                     | The port at which to listen for RPC requests.                                                                                                  |
+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+
| log_level                    | ``chaind``'s log level. Can be one of the following: ``trace``, ``debug``, ``info``, ``warn``, ``error``, ``crit``.                            |
+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+
| ``[log_auditor]``.log_file   | The location of ``chaind``'s audit log file                                                                                                    |
+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+
| ``[redis]``.url              | URL to an instance of Redis.                                                                                                                   |
+------------------------------+------------------------------------------------------------------------------------------------------------------------------------------------+