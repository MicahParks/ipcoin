CREATE TABLE transfer
(
    created   TIMESTAMPTZ NOT NULL,
    id        UUID PRIMARY KEY,
    sender    INET        NOT NULL,
    recipient INET        NOT NULL,
    amount    BIGINT      NOT NULL
);
CREATE INDEX on transfer (sender);
CREATE INDEX on transfer (recipient);
CREATE INDEX on transfer (created DESC);

CREATE TABLE comment
(
    created TIMESTAMPTZ NOT NULL,
    id      UUID PRIMARY KEY,
    address INET        NOT NULL,
    message TEXT        NOT NULL
);
CREATE INDEX on comment (address);
CREATE INDEX on comment (created DESC);

CREATE TABLE comment_moderation
(
    created    TIMESTAMPTZ NOT NULL,
    id         UUID PRIMARY KEY,
    comment_id UUID        NOT NULL REFERENCES comment (id) ON DELETE CASCADE,
    censored   BOOLEAN     NOT NULL DEFAULT FALSE,
    note       TEXT        NOT NULL
);
CREATE INDEX on comment_moderation (comment_id);

CREATE
MATERIALIZED VIEW leaderboard_glance AS
WITH comments AS (SELECT address, COUNT(*) AS comment_count
                  FROM comment
                  GROUP BY address),
     transfers AS (SELECT address,
                          COUNT(*)    AS transfer_count,
                          SUM(amount) AS balance_diff
                   FROM (SELECT recipient AS address, amount
                         FROM transfer
                         UNION ALL
                         SELECT sender AS address, -amount
                         FROM transfer) AS flat
                   GROUP BY address)
SELECT COALESCE(t.address, c.address) AS address,
       COALESCE(t.balance_diff, 0)    AS balance_diff,
       COALESCE(c.comment_count, 0)   AS comment_count,
       COALESCE(t.transfer_count, 0)  AS transfer_count
FROM transfers t
         FULL OUTER JOIN comments c ON t.address = c.address; -- Change to a LEFT JOIN if performance becomes an issue.
CREATE UNIQUE INDEX on leaderboard_glance (address);
REFRESH
MATERIALIZED VIEW CONCURRENTLY leaderboard_glance;
CREATE INDEX on leaderboard_glance (balance_diff DESC, address ASC) WHERE balance_diff > 0;
CREATE INDEX on leaderboard_glance (comment_count DESC, address ASC) WHERE comment_count > 0;
CREATE INDEX on leaderboard_glance (transfer_count DESC, address ASC) WHERE transfer_count > 0;
