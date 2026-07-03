ALTER TABLE `nodes`
    ADD COLUMN `target_node_id` bigint NULL COMMENT '入口节点绑定的落地节点ID' AFTER `is_hidden`;

CREATE INDEX `idx_nodes_target_node_id` ON `nodes` (`target_node_id`);
