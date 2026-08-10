"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[443],{

/***/ 2700
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_api_reference_md_964_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-api-reference-md-964.json
const site_docs_api_reference_md_964_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"api-reference","title":"API Reference","description":"s3proxy exposes an S3-compatible HTTP API rather than a custom JSON API.","source":"@site/docs/api-reference.md","sourceDirName":".","slug":"/api-reference","permalink":"/docs/api-reference","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":5,"frontMatter":{"sidebar_position":5},"sidebar":"docsSidebar","previous":{"title":"Request Examples","permalink":"/docs/request-examples"},"next":{"title":"Operations","permalink":"/docs/operations"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/api-reference.md


const frontMatter = {
	sidebar_position: 5
};
const contentTitle = 'API Reference';

const assets = {

};



const toc = [{
  "value": "Supported Operations",
  "id": "supported-operations",
  "level": 2
}, {
  "value": "Addressing Modes",
  "id": "addressing-modes",
  "level": 2
}, {
  "value": "Authentication Modes",
  "id": "authentication-modes",
  "level": 2
}, {
  "value": "Request Classification",
  "id": "request-classification",
  "level": 2
}, {
  "value": "Routing-Specific Behavior",
  "id": "routing-specific-behavior",
  "level": 2
}, {
  "value": "<code>ListBuckets</code>",
  "id": "listbuckets",
  "level": 2
}, {
  "value": "<code>ListObjectsV2</code>",
  "id": "listobjectsv2",
  "level": 2
}, {
  "value": "Failover Rules",
  "id": "failover-rules",
  "level": 2
}, {
  "value": "Fan-Out Writes",
  "id": "fan-out-writes",
  "level": 2
}, {
  "value": "Outbound Signing",
  "id": "outbound-signing",
  "level": 2
}, {
  "value": "Error Behavior",
  "id": "error-behavior",
  "level": 2
}, {
  "value": "Health Endpoints",
  "id": "health-endpoints",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    p: "p",
    table: "table",
    tbody: "tbody",
    td: "td",
    th: "th",
    thead: "thead",
    tr: "tr",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "api-reference",
        children: "API Reference"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " exposes an S3-compatible HTTP API rather than a custom JSON API."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This page focuses on the proxy-facing contract, supported S3 operations, and the behavior that is specific to the proxy."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "supported-operations",
      children: "Supported Operations"
    }), "\n", (0,jsx_runtime.jsxs)(_components.table, {
      children: [(0,jsx_runtime.jsx)(_components.thead, {
        children: (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.th, {
            children: "Operation"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Supported"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Notes"
          })]
        })
      }), (0,jsx_runtime.jsxs)(_components.tbody, {
        children: [(0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "GetObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Reads from one effective destination"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Reads from one effective destination"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "PutObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Can fan out with ", (0,jsx_runtime.jsx)(_components.code, {
              children: "dispatch = \"all\""
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "DeleteObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Can fan out with ", (0,jsx_runtime.jsx)(_components.code, {
              children: "dispatch = \"all\""
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadBucket"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Route-selected backend"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListObjectsV2"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Uses one effective backend only"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListObjectsV1"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Returns ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListBuckets"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Proxy-defined virtual bucket list"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "CopyObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Returns ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: "Multipart upload operations"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Return ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        })]
      })]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "addressing-modes",
      children: "Addressing Modes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy accepts:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "path-style addressing"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "virtual-hosted addressing"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The listener decides which forms are enabled."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "authentication-modes",
      children: "Authentication Modes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported inbound auth modes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "none"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["With ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), ", the proxy verifies the inbound S3 SigV4 signature against statically configured clients."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "request-classification",
      children: "Request Classification"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routing uses S3 operation classification, not only the HTTP method."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Examples:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "GET /bucket/key"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "GetObject"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "HEAD /bucket"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadBucket"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "GET /?list-type=2"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListObjectsV2"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["That classification is what route ", (0,jsx_runtime.jsx)(_components.code, {
        children: "operations = [...]"
      }), " filters use."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "routing-specific-behavior",
      children: "Routing-Specific Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routes are evaluated in config order and may stop or continue after a match."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important behavior:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"first\""
        }), " uses one destination"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), " replays supported writes to all destinations"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads never fan out in v1"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "read_preference"
        }), " chooses the effective backend for reads when multiple destinations are configured"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "listbuckets",
      children: (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " is virtual."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The proxy returns buckets defined in ", (0,jsx_runtime.jsx)(_components.code, {
        children: "bucket"
      }), " blocks and filtered by the authenticated client's ", (0,jsx_runtime.jsx)(_components.code, {
        children: "visible_buckets"
      }), " policy. It does not call the backend to discover buckets."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "listobjectsv2",
      children: (0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      }), " is forwarded to one selected backend."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy does not merge listing results or pagination tokens across multiple backends in v1."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "failover-rules",
      children: "Failover Rules"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "read_preference = \"ordered_failover\""
      }), ", failover happens only on:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "transport errors"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "request timeouts"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["upstream ", (0,jsx_runtime.jsx)(_components.code, {
          children: "5xx"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Failover does not happen on:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchKey"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchBucket"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "fan-out-writes",
      children: "Fan-Out Writes"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"all\""
      }), ":"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "PutObject"
        }), " is supported"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "DeleteObject"
        }), " is supported"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["the request body is buffered in memory so it can be replayed, bounded by ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_max_bytes"
        }), " per request and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_aggregate_max_bytes"
        }), " across the process"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "if any destination fails, the request fails overall"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "upstream HTTP failures preserve the primary upstream error response when available"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "transport or replay failures return a proxy-generated failure"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "fan-out is not transactional; a destination that succeeds before another destination fails is not rolled back"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["oversized replay attempts fail with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "413 EntityTooLarge"
        }), "; aggregate replay-budget exhaustion fails with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "503 SlowDown"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For writes matched by multiple routes through ", (0,jsx_runtime.jsx)(_components.code, {
        children: "on_match = \"continue\""
      }), ", every matched route must also succeed. A later route failure is returned as failure rather than hiding behind an earlier success."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "outbound-signing",
      children: "Outbound Signing"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy terminates and rebuilds requests before forwarding them."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Outbound S3 requests are signed with the destination backend credentials, not the inbound client credentials."
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For outbound SigV4, the proxy uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "UNSIGNED-PAYLOAD"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Content-Length"
      }), " must be set before signing."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "error-behavior",
      children: "Error Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Representative error rules:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["unsupported operations return S3-compatible ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NotImplemented"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "route misses return standard S3-compatible error responses"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "upstream backend failures propagate as proxy-mediated S3 responses"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "multi-destination write failures are surfaced as failures, not partial success"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "multi-route write failures are surfaced as failures, not partial success"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "health-endpoints",
      children: "Health Endpoints"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "GET /healthz"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "GET /readyz"
      }), " are local process endpoints. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/readyz"
      }), " means the proxy process is serving requests; it does not poll configured backends or report destination health."]
    })]
  });
}
function MDXContent(props = {}) {
  const {wrapper: MDXLayout} = {
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return MDXLayout ? (0,jsx_runtime.jsx)(MDXLayout, {
    ...props,
    children: (0,jsx_runtime.jsx)(_createMdxContent, {
      ...props
    })
  }) : _createMdxContent(props);
}



/***/ },

/***/ 1982
(__unused_webpack___webpack_module__, __webpack_exports__, __webpack_require__) {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   R: () => (/* binding */ useMDXComponents),
/* harmony export */   x: () => (/* binding */ MDXProvider)
/* harmony export */ });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(489);
/**
 * @import {MDXComponents} from 'mdx/types.js'
 * @import {Component, ReactElement, ReactNode} from 'react'
 */

/**
 * @callback MergeComponents
 *   Custom merge function.
 * @param {Readonly<MDXComponents>} currentComponents
 *   Current components from the context.
 * @returns {MDXComponents}
 *   Additional components.
 *
 * @typedef Props
 *   Configuration for `MDXProvider`.
 * @property {ReactNode | null | undefined} [children]
 *   Children (optional).
 * @property {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @property {boolean | null | undefined} [disableParentContext=false]
 *   Turn off outer component context (default: `false`).
 */



/** @type {Readonly<MDXComponents>} */
const emptyComponents = {}

const MDXContext = react__WEBPACK_IMPORTED_MODULE_0__.createContext(emptyComponents)

/**
 * Get current components from the MDX Context.
 *
 * @param {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @returns {MDXComponents}
 *   Current components.
 */
function useMDXComponents(components) {
  const contextComponents = react__WEBPACK_IMPORTED_MODULE_0__.useContext(MDXContext)

  // Memoize to avoid unnecessary top-level context changes
  return react__WEBPACK_IMPORTED_MODULE_0__.useMemo(
    function () {
      // Custom merge via a function prop
      if (typeof components === 'function') {
        return components(contextComponents)
      }

      return {...contextComponents, ...components}
    },
    [contextComponents, components]
  )
}

/**
 * Provider for MDX context.
 *
 * @param {Readonly<Props>} properties
 *   Properties.
 * @returns {ReactElement}
 *   Element.
 * @satisfies {Component}
 */
function MDXProvider(properties) {
  /** @type {Readonly<MDXComponents>} */
  let allComponents

  if (properties.disableParentContext) {
    allComponents =
      typeof properties.components === 'function'
        ? properties.components(emptyComponents)
        : properties.components || emptyComponents
  } else {
    allComponents = useMDXComponents(properties.components)
  }

  return react__WEBPACK_IMPORTED_MODULE_0__.createElement(
    MDXContext.Provider,
    {value: allComponents},
    properties.children
  )
}


/***/ }

}]);