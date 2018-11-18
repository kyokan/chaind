Installation
============

From Package
------------

The fastest way to install ``chaind`` is to install it via your OS's package manager. We currently have Debian packages
available, with RPMs coming soon. To install from our Debian repository, follow these steps:

**1. Add Kyokan's signing key**

First, you'll need to instruct your package manager to trust Kyokan's PGP key. Our PGP key is stored on MIT's keyserver,
and has the fingerprint ``A27E D5CE 49FA EB7D E0AA  50D4 23D7 FC00 2105 25``.

You'll only need to perform this step once per machine. Run the command below to use our key:

.. code-block:: bash

    sudo apt-key adv --keyserver pgp.mit.edu --recv-keys 21052518

You should receive output that looks like this:

.. code-block:: text

    Executing: /tmp/apt-key-gpghome.MZuO4hoFiT/gpg.1.sh --keyserver pgp.mit.edu --recv-keys 21052518
    gpg: key 23D7FC0021052518: public key "Kyokan, LLC OSS Signing Key <mslipper@kyokan.io>" imported
    gpg: Total number processed: 1
    gpg:               imported: 1

**2. Add Kyokan's Debian repository to your sources.list**

Run the following command to add Kyokan's Debian repository to your ``sources.list`` file:

.. code-block:: bash

    echo "deb https://dl.bintray.com/kyokan/oss-deb any main" | sudo tee -a /etc/apt/sources.list

This also only needs to be done once per machine.

**3. Install chaind**

Now you're ready to install ``chaind``. To do so, run these commands:

.. code-block:: bash

    sudo apt-get update
    sudo apt-get install chaind

From Source
-----------

To build and install ``chaind`` from source, you'll need to first install the following prerequisites:

1. ``go`` version 1.10 or higher
2. ``make``
3. ``dep``
4. ``git``

Now, get the chaind source by running ``go get -u github.com/kyokan/chaind`` and ``cd`` into the source directory. You
can now build and install ``chaind`` with the following commands:

.. code-block:: bash

    make deps
    make build
    make install-global

``make install-global`` will place the ``chaind`` binary in ``/usr/bin``. You'll need to create a configuration file
manually.

Next Steps
----------

Now, you're ready to configure your ``chaind`` instance. See our Configuration page for more information.