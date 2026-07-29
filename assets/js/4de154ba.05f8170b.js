"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[81],{

/***/ 4852
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_operations_md_4de_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-operations-md-4de.json
const site_docs_operations_md_4de_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"operations","title":"Operations","description":"This page covers the commands and runtime behavior that matter most when working on or operating s3proxy.","source":"@site/docs/operations.md","sourceDirName":".","slug":"/operations","permalink":"/docs/operations","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":7,"frontMatter":{"sidebar_position":7},"sidebar":"docsSidebar","previous":{"title":"API Reference","permalink":"/docs/api-reference"},"next":{"title":"Deployment","permalink":"/docs/deployment"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/operations.md


const frontMatter = {
	sidebar_position: 7
};
const contentTitle = 'Operations';

const assets = {

};



const toc = [{
  "value": "Build And Run",
  "id": "build-and-run",
  "level": 2
}, {
  "value": "Environment Variables",
  "id": "environment-variables",
  "level": 2
}, {
  "value": "Docker",
  "id": "docker",
  "level": 2
}, {
  "value": "Unit Test And Validation Commands",
  "id": "unit-test-and-validation-commands",
  "level": 2
}, {
  "value": "Integration Tests",
  "id": "integration-tests",
  "level": 2
}, {
  "value": "Sandbox Commands",
  "id": "sandbox-commands",
  "level": 2
}, {
  "value": "Runtime Behavior",
  "id": "runtime-behavior",
  "level": 2
}, {
  "value": "Logging And Diagnostics",
  "id": "logging-and-diagnostics",
  "level": 2
}, {
  "value": "Suggested Local Checklist",
  "id": "suggested-local-checklist",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    ol: "ol",
    p: "p",
    pre: "pre",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "operations",
        children: "Operations"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This page covers the commands and runtime behavior that matter most when working on or operating ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "build-and-run",
      children: "Build And Run"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Common commands:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make build\nmake build-all\nmake run CONFIG=path/to/config.hcl\nmake validate CONFIG=path/to/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The standard binary entrypoints are:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy serve --config /etc/s3proxy/config.hcl\ns3proxy validate --config /etc/s3proxy/config.hcl\ns3proxy version\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "make build"
      }), " produces ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dist/s3proxy"
      }), " for the host platform with CGO disabled."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "environment-variables",
      children: "Environment Variables"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The config loader inlines ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env(\"VAR\")"
      }), " before HCL parsing."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If you run locally with a ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), " file, load it before invoking the proxy:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "set -a; . ./.env; set +a\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "docker",
      children: "Docker"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Build and run with the repo helpers:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make docker-build\nmake docker-run CONFIG=path/to/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The image mounts the config file and runs the same CLI entrypoint as the local binary."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "unit-test-and-validation-commands",
      children: "Unit Test And Validation Commands"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make vet\nmake test\nmake test-race\nmake cover\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The standard local sanity check is:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make vet test\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "There is no separate typecheck target. A successful Go build is the typecheck."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "integration-tests",
      children: "Integration Tests"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The integration suite is build-tagged ", (0,jsx_runtime.jsx)(_components.code, {
        children: "integration"
      }), " and is skipped by ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make test"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Canonical one-shot flow:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "cp .env.example .env\nmake sandbox-integration-up\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Iterative flow:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make sandbox-up DAEMON=true\nmake build\nmake test-integration\nmake test-integration-race\nmake sandbox-down\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The sandbox stack exercises the proxy end to end against MinIO and SeaweedFS."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "sandbox-commands",
      children: "Sandbox Commands"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Useful sandbox helpers:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make sandbox-up\nmake sandbox-down\nmake sandbox-destroy\nmake sandbox-reset\nmake sandbox-logs\nmake sandbox-logs-follow\nmake sandbox-ps\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The sandbox compose file lives at ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox/docker-compose.yml"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "runtime-behavior",
      children: "Runtime Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important operational behaviors:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "only one listener is supported"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "config changes require a restart; there is no hot reload in v1"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["request bodies that need replay are buffered in memory up to ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_max_bytes"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads use one effective backend even when a route has multiple destinations"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["target ", (0,jsx_runtime.jsx)(_components.code, {
          children: "timeout"
        }), " directly affects failover timing for ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If the replay limit is exceeded, the proxy returns ", (0,jsx_runtime.jsx)(_components.code, {
        children: "413 EntityTooLarge"
      }), " instead of attempting the upstream request."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "logging-and-diagnostics",
      children: "Logging And Diagnostics"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy validate --config ..."
      }), " before rollouts to catch configuration errors such as:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "invalid parser definitions"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "unknown target or route references"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "unsupported operation names"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "invalid auth mode or missing clients"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For integration troubleshooting, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make sandbox-logs"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make sandbox-logs-follow"
      }), " are the fastest way to inspect backend behavior."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "suggested-local-checklist",
      children: "Suggested Local Checklist"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Load environment variables used by ", (0,jsx_runtime.jsx)(_components.code, {
          children: "env(\"...\")"
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "make vet test"
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy validate --config ..."
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Start the proxy and confirm ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), " and one object read/write path."]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "If using replication or failover, run the integration suite against the sandbox."
      }), "\n"]
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