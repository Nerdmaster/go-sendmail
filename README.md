# go-sendmail

/sbin/sendmail replacement to use customized SMTP settings based on various header fields

This project aims to deliver a very simple binary capable of acting as (or
replacing) the system `sendmail` binary in simple use-cases.  The main use-case
is a PHP server which needs to deliver email (such as a Wordpress site), but
doesn't have any kind of mail program installed.

The binary doesn't try to do everything sendmail does.  It is only meant to
handle the simplest cases, like PHP, in order to fill the gap on minimal
servers.

This is a work in progress.  For now, look at the config example to get an idea
how it works.
