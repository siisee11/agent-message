#!/usr/bin/env node

import { messageJsonRenderCatalog } from '../web/src/components/messageJsonRenderCatalog.shared.mjs'

process.stdout.write(messageJsonRenderCatalog.prompt())
