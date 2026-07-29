"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[177],{

/***/ 8634
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_providers_and_routing_md_80a_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-providers-and-routing-md-80a.json
const site_docs_providers_and_routing_md_80a_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"providers-and-routing","title":"Routing and Rewrites","description":"s3proxy routes requests from a normalized S3 request context, not just from the raw HTTP method.","source":"@site/docs/providers-and-routing.md","sourceDirName":".","slug":"/providers-and-routing","permalink":"/docs/providers-and-routing","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":4,"frontMatter":{"sidebar_position":4},"sidebar":"docsSidebar","previous":{"title":"Config Examples","permalink":"/docs/config-examples"},"next":{"title":"Request Examples","permalink":"/docs/request-examples"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/providers-and-routing.md


const frontMatter = {
	sidebar_position: 4
};
const contentTitle = 'Routing and Rewrites';

const assets = {

};



const toc = [{
  "value": "Addressing Modes",
  "id": "addressing-modes",
  "level": 2
}, {
  "value": "Parser Types",
  "id": "parser-types",
  "level": 2
}, {
  "value": "Route Evaluation Order",
  "id": "route-evaluation-order",
  "level": 2
}, {
  "value": "Dispatch Modes",
  "id": "dispatch-modes",
  "level": 2
}, {
  "value": "Read Preference",
  "id": "read-preference",
  "level": 2
}, {
  "value": "Rewrite Rules",
  "id": "rewrite-rules",
  "level": 2
}, {
  "value": "Strict Prefix Matching",
  "id": "strict-prefix-matching",
  "level": 2
}, {
  "value": "Virtual Buckets And Routing",
  "id": "virtual-buckets-and-routing",
  "level": 2
}, {
  "value": "Common Patterns",
  "id": "common-patterns",
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
    pre: "pre",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "routing-and-rewrites",
        children: "Routing and Rewrites"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " routes requests from a normalized S3 request context, not just from the raw HTTP method."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "That matters because the proxy can match on path, bucket, host, and operation, then rewrite the outgoing bucket and key before signing the request for the destination backend."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "addressing-modes",
      children: "Addressing Modes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy supports both S3 addressing styles:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["path-style requests such as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "/bucket/key"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["virtual-hosted requests such as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket.s3proxy.example.com/key"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Enable them in the listener:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = true\n    host_suffixes  = [\"s3proxy.example.com\"]\n  }\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "parser-types",
      children: "Parser Types"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routes reference a parser. Supported parser kinds are:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "path_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_exact"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_regex"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "host_suffix"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Examples:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "parser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nparser \"bucket_regex\" \"tenant_logs\" {\n  pattern = \"^tenant-(?P<tenant>[a-z0-9-]+)-logs$\"\n}\n\nparser \"host_suffix\" \"public_hosts\" {\n  suffix = \"s3proxy.example.com\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "bucket_regex"
      }), " can capture named groups, which are then exposed to rewrite templates."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "route-evaluation-order",
      children: "Route Evaluation Order"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routes are evaluated in config order."
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Each route has an ", (0,jsx_runtime.jsx)(_components.code, {
        children: "on_match"
      }), " policy:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "stop"
        }), ": stop evaluating routes after the match"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "continue"
        }), ": keep evaluating later routes and collect more matches"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This lets you compose behavior. For example, one ", (0,jsx_runtime.jsx)(_components.code, {
        children: "PutObject"
      }), " can be applied to multiple matching routes if the earlier match uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "continue"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "dispatch-modes",
      children: "Dispatch Modes"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Each route also has a ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch"
      }), " policy:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        }), ": use a single destination"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "all"
        }), ": send the write to every destination"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"all\""
      }), " is for multi-destination writes such as replication."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "In v1:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "writes can fan out"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads never fan out"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "if any destination fails during a fan-out write, the request fails"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "the client gets the primary upstream response body rather than a generic proxy error body"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "read-preference",
      children: "Read Preference"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["When a route has more than one destination, reads choose one effective backend using ", (0,jsx_runtime.jsx)(_components.code, {
        children: "read_preference"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "random"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "hash"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ordered_failover"
      }), " is the safest choice when you want a preferred backend with a backup."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Failover happens only on:"
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
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "That behavior prevents the proxy from hiding routing mistakes or backend inconsistency."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "rewrite-rules",
      children: "Rewrite Rules"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "After route selection, the proxy can rewrite the request before it builds the outbound backend request."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported rewrite fields:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_path_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_key_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "prepend_key_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "key_template"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "route \"tenant_logs\" {\n  parser          = \"tenant_logs\"\n  operations      = [\"GetObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    bucket       = \"shared-logs\"\n    key_template = \"{{ .Captures.tenant }}/{{ .Key }}\"\n  }\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Template data includes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "Bucket"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "Key"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "Captures"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "strict-prefix-matching",
      children: "Strict Prefix Matching"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "path_prefix"
      }), " matching is intentionally strict."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If the prefix is ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/replica"
      }), ", a match happens only when:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "RawPath == \"/replica\""
        })
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "RawPath"
        }), " starts with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "\"/replica/\""
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "/replicate/..."
      }), " does not match ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/replica"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "virtual-buckets-and-routing",
      children: "Virtual Buckets And Routing"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " is proxy-defined, not upstream-defined."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "That means you can expose a clean bucket catalog even when the proxy is routing requests by path prefix or rewriting them into completely different backend buckets."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "bucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The visible bucket name presented to the client does not need to equal the backend bucket used after rewrite."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "common-patterns",
      children: "Common Patterns"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Single backend, path-based view:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "parser \"path_prefix\" \"primary_prefix\" {\n  prefix = \"/primary\"\n}\n\nroute \"primary_only\" {\n  parser          = \"primary_prefix\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"HeadBucket\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix = \"/primary\"\n    bucket            = \"testbucket\"\n  }\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Replicated writes:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "route \"replicate_rw\" {\n  parser          = \"replicate_prefix\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\"]\n  destinations    = [\"primary\", \"replica\"]\n  dispatch        = \"all\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Preferred backend with failover:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "route \"replica_failover_read\" {\n  parser          = \"failover_prefix\"\n  operations      = [\"GetObject\", \"HeadObject\"]\n  destinations    = [\"missing\", \"replica\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"ordered_failover\"\n}\n"
      })
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