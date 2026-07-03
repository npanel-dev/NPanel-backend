DROP INDEX `idx_nodes_target_node_id` ON `nodes`;

ALTER TABLE `nodes`
    DROP COLUMN `target_node_id`;
